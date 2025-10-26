package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"pbl/shared"
	"pbl/server/game"
	"pbl/server/models"

	"github.com/nats-io/nats.go"
)

// HandleGlobalGameMessage processa jogadas de cartas para salas globais
func HandleGlobalGameMessage(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
	var gameMsg shared.GameMessage
	if err := json.Unmarshal(request.Payload, &gameMsg); err != nil {
		log.Println("[Global] Erro ao decodificar GameMessage:", err)
		return
	}

	log.Printf("[Global] CARTA RECEBIDA no servidor %d: %s", server.ID, gameMsg.RoomID)

	// Acessa a sala global pelo FSM
	server.FSM.GlobalRoomsMu.Lock()
	room, exists := server.FSM.GlobalRooms[gameMsg.RoomID]
	server.FSM.GlobalRoomsMu.Unlock()

	if !exists {
		// Sala não encontrada: determinar host pelo nome da sala
		var hostServerID int
		_, err := fmt.Sscanf(gameMsg.RoomID, "global-%d-vs-", &hostServerID)
		if err != nil {
			log.Printf("[Global] Erro ao extrair ID do host da sala %s: %v", gameMsg.RoomID, err)
			return
		}

		if hostServerID == server.ID {
			log.Printf("[Global] Sala não encontrada, mas este servidor (%d) é o host. Ignorando para evitar loop.", server.ID)
			return
		}

		log.Printf("[Global] Sala %s não encontrada localmente. Encaminhando para host %d...", gameMsg.RoomID, hostServerID)
		hostTopic := fmt.Sprintf("server.%d.requests", hostServerID)
		nc.Publish(hostTopic, msg.Data)
		return
	}

	// Sala encontrada: processa a carta
	processGlobalCard(room, gameMsg, nc)
}

// processGlobalCard processa a carta jogada, envia para o adversário e resolve rodada se necessário
func processGlobalCard(room *shared.GameRoom, gameMsg shared.GameMessage, nc *nats.Conn) {
	// Inicializa mapa de cartas se necessário
	if room.PlayersCards == nil {
		room.PlayersCards = make(map[string]shared.Card)
	}

	// Decodifica a carta jogada
	var card shared.Card
	if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
		log.Println("[Global] Erro ao decodificar carta:", err)
		return
	}

	room.PlayersCards[gameMsg.From] = card
	log.Printf("[Global] Carta recebida de %s: %+v", gameMsg.From, card)

	// Determina próximo turno
	var nextTurn string
	if gameMsg.From == room.Player1.UserId {
		nextTurn = room.Player2.UserId
	} else {
		nextTurn = room.Player1.UserId
	}
	room.Turn = nextTurn

	// Envia carta ao adversário
	opponentID := room.Player1.UserId
	if gameMsg.From == room.Player1.UserId {
		opponentID = room.Player2.UserId
	}

	turnMsg := shared.GameMessage{
		Type: "PLAY_CARD",
		From: gameMsg.From,
		Data: gameMsg.Data,
		Turn: nextTurn,
	}
	dataTurn, _ := json.Marshal(turnMsg)

	nc.Publish(fmt.Sprintf("client.%s.inbox", opponentID), dataTurn)
	log.Printf("[Global] Jogada encaminhada ao oponente %s", opponentID)

	// Se ambos já jogaram, calcula resultado
	if len(room.PlayersCards) == 2 {
		cardP1 := room.PlayersCards[room.Player1.UserId]
		cardP2 := room.PlayersCards[room.Player2.UserId]

		resultP1 := game.CheckWinner(cardP1, cardP2)
		NotifyResult(nc, room, resultP1)

		// Limpa cartas da rodada
		room.PlayersCards = make(map[string]shared.Card)
	}
}

// NotifyResult envia o resultado da rodada aos jogadores
func NotifyResult(nc *nats.Conn, room *shared.GameRoom, resultP1 string) {
	var resultP2 string
	switch resultP1 {
	case "GANHOU":
		resultP2 = "PERDEU"
		room.Winner = room.Player1
	case "PERDEU":
		resultP2 = "GANHOU"
		room.Winner = room.Player2
	case "EMPATE":
		resultP2 = "EMPATE"
		room.Winner = nil
	}

	// Notifica Player1
	msgP1 := shared.GameMessage{
		Type:   "ROUND_RESULT",
		From:   "SERVER",
		Result: resultP1,
		Winner: room.Winner,
	}
	dataP1, _ := json.Marshal(msgP1)
	nc.Publish(fmt.Sprintf("client.%s.inbox", room.Player1.UserId), dataP1)

	// Notifica Player2
	msgP2 := shared.GameMessage{
		Type:   "ROUND_RESULT",
		From:   "SERVER",
		Result: resultP2,
		Winner: room.Winner,
	}
	dataP2, _ := json.Marshal(msgP2)
	nc.Publish(fmt.Sprintf("client.%s.inbox", room.Player2.UserId), dataP2)

	// Limpa cartas da rodada
	room.PlayersCards = make(map[string]shared.Card)
}
