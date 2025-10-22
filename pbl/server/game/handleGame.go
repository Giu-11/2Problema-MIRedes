package game

import (
	"log"
	"fmt"
	"encoding/json"
	
	"pbl/shared"

    "github.com/nats-io/nats.go"
)
func HandleIncomingGameMessage(msgData []byte) {
    //log.Printf("Mensagem bruta recebida: %s", string(msgData))
    
    var msg shared.GameMessage
    if err := json.Unmarshal(msgData, &msg); err != nil {
        log.Println("Erro ao decodificar GameMessage:", err)
        return
    }
    
    //DEBUG: Print da struct completa
    //log.Printf("GameMessage decodificado: %+v", msg)
    //log.Printf("Tipo de mensagem: '%s' (len=%d)", msg.Type, len(msg.Type))
    
    switch msg.Type {
    case "PLAY_CARD":
        var card shared.Card
        if err := json.Unmarshal(msg.Data, &card); err != nil {
            log.Println("Erro ao decodificar carta jogada:", err)
            return
        }
        fmt.Printf("\nO oponente jogou: %s (%s)\n", card.Element, card.Type)
    case "MATCH":
        var opponent shared.User
        if err := json.Unmarshal(msg.Data, &opponent); err != nil {
            log.Println("Erro ao decodificar dados do adversário:", err)
            return
        }
        fmt.Printf("\nPartida iniciada! Oponente: %s\n", opponent.UserName)
    case "GAME_OVER":
        //tem que ver isso aqui
    default:
        log.Printf("Mensagem desconhecida recebida - Type: '%s'", msg.Type)
    }
}

func NotifyResult(nc *nats.Conn, room *shared.GameRoom, resultP1 string) {
	// Define o resultado do player 2 baseado no player 1
	var resultP2 string
	switch resultP1 {
	case "GANHOU":
		resultP2 = "PERDEU"
		room.Winner = room.Player1
		fmt.Println("P2 perdeu tadinho")
	case "PERDEU":
		resultP2 = "GANHOU"
		room.Winner = room.Player2
		fmt.Println("P2 ganhou")
	case "EMPATE":
		resultP2 = "EMPATE"
		room.Winner = nil
		fmt.Println("Emapntou")
	}

	//Notifica Player1
	msgP1 := shared.GameMessage{
		Type:   "ROUND_RESULT",
		From:   "SERVER",
		Result: resultP1,
		Winner: room.Winner,
	}
	dataP1, _ := json.Marshal(msgP1)
	nc.Publish(fmt.Sprintf("client.%s.inbox", room.Player1.UserId), dataP1)

	//Notifica Player2
	msgP2 := shared.GameMessage{
		Type:   "ROUND_RESULT",
		From:   "SERVER",
		Result: resultP2,
		Winner: room.Winner,
	}
	dataP2, _ := json.Marshal(msgP2)
	nc.Publish(fmt.Sprintf("client.%s.inbox", room.Player2.UserId), dataP2)

	//Limpa as cartas da rodada
	room.PlayersCards = make(map[string]shared.Card)
}
