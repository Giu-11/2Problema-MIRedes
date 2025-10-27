package handlers

import (
	"fmt"
	"log"
	"time"
	"bytes"
	"net/http"
	"encoding/json"

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

	// MODIFICAÇÃO AQUI: Aguarda sala ser replicada com retry
	var room *shared.GameRoom
	var exists bool
	
	for i := 0; i < 10; i++ {
		server.FSM.GlobalRoomsMu.RLock()
		room, exists = server.FSM.GlobalRooms[gameMsg.RoomID]
		server.FSM.GlobalRoomsMu.RUnlock()
		
		if exists {
			log.Printf("[Global] ✅ Sala %s encontrada no servidor %d", gameMsg.RoomID, server.ID)
			break
		}
		
		log.Printf("[Global] ⏳ Sala %s não encontrada (tentativa %d/10). Aguardando replicação...", gameMsg.RoomID, i+1)
		time.Sleep(500 * time.Millisecond)
	}

	if !exists {
		log.Printf("[Global] ❌ Sala %s não encontrada no servidor %d após 5 tentativas", gameMsg.RoomID, server.ID)
		return
	}

	// Processa a carta do cliente local
	processClientCard(server, room, gameMsg, nc)
}

// processClientCard: processa carta do cliente LOCAL
func processClientCard(server *models.Server, room *shared.GameRoom, gameMsg shared.GameMessage, nc *nats.Conn) {
	var card shared.Card
	if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
		log.Println("[Global] Erro ao decodificar carta:", err)
		return
	}

	log.Printf("[Global] Cliente %s jogou %s", gameMsg.From, card.Element)

	// Determina oponente e próximo turno
	var opponentID string
	var opponentServerID int

	if gameMsg.From == room.Player1.UserId {
		opponentID = room.Player2.UserId
		opponentServerID = room.Server2ID
	} else {
		opponentID = room.Player1.UserId
		opponentServerID = room.Server1ID
	}

	// 1. Notifica o oponente
	turnMsg := shared.GameMessage{
		Type:   "PLAY_CARD",
		From:   gameMsg.From,
		RoomID: room.ID,
		Data:   gameMsg.Data,
		Turn:   opponentID,
	}

	if opponentServerID != server.ID {
		sendCardToOpponentServer(opponentServerID, opponentID, turnMsg)
	} else {
		dataTurn, _ := json.Marshal(turnMsg)
		clientTopic := fmt.Sprintf("server.%d.client.%s", server.ID, opponentID)
		nc.Publish(clientTopic, dataTurn)
		log.Printf("[Global] Oponente local notificado: %s", opponentID)
	}

	// 2. Envia carta ao HOST para cálculo
	if server.ID != room.ServerID {
		// Não sou o host, encaminho
		log.Printf("[Global] Encaminhando ao HOST (server%d)...", room.ServerID)
		forwardCardToHost(room.ServerID, gameMsg)
	} else {
		// Sou o host, armazeno e calculo
		processHostCard(server, room, gameMsg, nc)
	}
}

// processHostCard: HOST armazena carta e calcula resultado se possível
func processHostCard(server *models.Server, room *shared.GameRoom, gameMsg shared.GameMessage, nc *nats.Conn) {
	if room.PlayersCards == nil {
		room.PlayersCards = make(map[string]shared.Card)
	}

	var card shared.Card
	if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
		log.Println("[HOST] Erro ao decodificar carta:", err)
		return
	}

	log.Printf("[HOST] 🎯 Sala pointer: %p", room)
	log.Printf("[HOST] 🎯 PlayersCards pointer: %p", room.PlayersCards)

	room.PlayersCards[gameMsg.From] = card
	log.Printf("[HOST] Carta armazenada: %s -> %s (%d/2)", gameMsg.From, card.Element, len(room.PlayersCards))

	log.Printf("[HOST] 📝 Carta armazenada: %s -> %s", gameMsg.From, card.Element)
	log.Printf("[HOST] 📊 Estado atual: %v", room.PlayersCards)
	log.Printf("[HOST] 📊 Total: %d/2", len(room.PlayersCards))

	// Se ambos jogaram, calcula resultado
	if len(room.PlayersCards) == 2 {
		cardP1 := room.PlayersCards[room.Player1.UserId]
		cardP2 := room.PlayersCards[room.Player2.UserId]

		log.Printf("[HOST] %s: %s vs %s: %s", room.Player1.UserName, cardP1.Element, room.Player2.UserName, cardP2.Element)

		resultP1 := game.CheckWinner(cardP1, cardP2)


		// Notifica ambos
		notifyPlayerResult(server, nc, room, resultP1, room.Player1.UserId, room.Server1ID)
		notifyPlayerResult(server, nc, room, resultP1, room.Player2.UserId, room.Server2ID)

		// Limpa cartas e resultados
		room.PlayersCards = make(map[string]shared.Card)
		resultP1 = ""
		log.Printf("[HOST] Rodada finalizada")
	}
}


func sendCardToOpponentServer(serverID int, clientID string, gameMsg shared.GameMessage) {
	url := fmt.Sprintf("http://server%d:800%d/forward-card", serverID, serverID)
	
	payload := map[string]interface{}{
		"client_id": clientID,
		"game_msg":  gameMsg,
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[REST] Erro ao enviar carta: %v", err)
		return
	}
	defer resp.Body.Close()
	
	log.Printf("[REST] Carta enviada para server%d -> cliente %s", serverID, clientID)
}

func forwardCardToHost(hostServerID int, gameMsg shared.GameMessage) {
	url := fmt.Sprintf("http://server%d:800%d/forward-to-host", hostServerID, hostServerID)
	
	payload := map[string]interface{}{
		"game_msg": gameMsg,
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[REST] Erro ao encaminhar ao host: %v", err)
		return
	}
	defer resp.Body.Close()
	
	log.Printf("[REST] Carta encaminhada ao HOST server%d", hostServerID)
}

// Recebe carta de outro servidor para notificar cliente local
func HandleForwardCard(server *models.Server, nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ClientID string              `json:"client_id"`
			GameMsg  shared.GameMessage  `json:"game_msg"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		
		log.Printf("[REST] Carta recebida para cliente %s", payload.ClientID)
		
		// Publica para o cliente local via NATS
		data, _ := json.Marshal(payload.GameMsg)
		clientTopic := fmt.Sprintf("server.%d.client.%s", server.ID, payload.ClientID)
		
		if err := nc.Publish(clientTopic, data); err != nil {
			log.Printf("[REST] Erro ao publicar: %v", err)
			http.Error(w, "Failed to forward", http.StatusInternalServerError)
			return
		}
		
		log.Printf("[REST] Carta encaminhada ao cliente %s via NATS", payload.ClientID)
		w.WriteHeader(http.StatusOK)
	}
}

// HandleForwardToHost: HOST recebe carta de servidor remoto
func HandleForwardToHost(server *models.Server, nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			GameMsg shared.GameMessage `json:"game_msg"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		
		log.Printf("[REST-HOST] Carta recebida: sala %s, jogador %s", 
			payload.GameMsg.RoomID, payload.GameMsg.From)
		
		server.FSM.GlobalRoomsMu.Lock()
		room, exists := server.FSM.GlobalRooms[payload.GameMsg.RoomID]
		server.FSM.GlobalRoomsMu.Unlock()
		
		if !exists {
			log.Printf("[REST-HOST] Sala não encontrada")
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		
		// Processa como host
		processHostCard(server, room, payload.GameMsg, nc)
		
		w.WriteHeader(http.StatusOK)
	}
}

// HandleForwardResult: encaminha resultado para cliente remoto (reutiliza HandleForwardCard)
func notifyPlayerResult(server *models.Server, nc *nats.Conn, room *shared.GameRoom, result string, playerID string, playerServerID int) {
	var winner *shared.User
	switch result {
	case "GANHOU":
		winner = room.Player1
	case "PERDEU":
		winner = room.Player2
	case "EMPATE":
		winner = nil
	}

	resultMsg := shared.GameMessage{
		Type:   "ROUND_RESULT",
		RoomID: room.ID,
		Winner: winner,
	}

	if playerServerID != server.ID {
		// Envia via REST
		url := fmt.Sprintf("http://server%d:800%d/forward-result", playerServerID, playerServerID)
		
		payload := map[string]interface{}{
			"client_id": playerID,
			"game_msg":  resultMsg,
		}
		
		jsonData, _ := json.Marshal(payload)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("[REST] Erro ao enviar resultado: %v", err)
			return
		}
		defer resp.Body.Close()
		
		log.Printf("[REST] Resultado enviado para server%d -> cliente %s", playerServerID, playerID)
	} else {
		// Cliente local, usa NATS
		data, _ := json.Marshal(resultMsg)
		topic := fmt.Sprintf("server.%d.client.%s", server.ID, playerID)
		nc.Publish(topic, data)
		log.Printf("[HOST] Resultado enviado ao cliente local %s", playerID)
	}
}