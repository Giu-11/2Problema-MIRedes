package game

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"pbl/client/utils"
	"pbl/shared"

	"github.com/nats-io/nats.go"
)

type MatchInfo struct {
	Opponent shared.User
	Room     shared.GameRoom
}

//preciso da parte das cartas primeiro pra continuar implementando isso aqui
func ChooseCard(user shared.User) (shared.Card, bool) {
	
	utils.ListCardsDeck(&user)
	cards := user.Deck

	fmt.Print("Insira a carta desejada (0 para sair): ")

	choice := utils.ReadLineSafe()
	choiceInt, err := strconv.Atoi(choice) 

	if choiceInt == 0 {
		user.Status = "available"
    	return shared.Card{}, false //saída voluntária do usuário
	}

	if err != nil || choiceInt < 1 || choiceInt > len(cards) {
		fmt.Println("Escolha inválida!")
		return shared.Card{}, false
	}

	selected := cards[choiceInt-1]
	return selected, true
}

/*
//enviado em broadcast 
func SendCardPlay(nc *nats.Conn, room *shared.GameRoom, fromUserID string, card shared.Card) {
    dataBytes, _ := json.Marshal(card)

    //Envia a carta para ambos, mas mantém o turno como o jogador atual
    for _, player := range []*shared.User{room.Player1, room.Player2} {
        msg := shared.GameMessage{
            Type: "PLAY_CARD",
            Data: dataBytes,
            From: fromUserID,
            Turn: fromUserID, 
        }

        bytes, _ := json.Marshal(msg)
        topic := fmt.Sprintf("client.%s.inbox", player.UserId)
        nc.Publish(topic, bytes)
    }
}*/


func SendCardPlay(nc *nats.Conn, room *shared.GameRoom, fromUserID string, card shared.Card) {
	dataBytes, _ := json.Marshal(card)

	gameMsg := shared.GameMessage{
		Type:   "PLAY_CARD",
		From:   fromUserID,
		RoomID: room.ID,
		Data:   dataBytes,
	}

	payload, _ := json.Marshal(gameMsg)

	req := shared.Request{
		ClientID: fromUserID,
		Action:   "GAME_MESSAGE",
		Payload:  payload,
	}

	reqBytes, _ := json.Marshal(req)
	topic := fmt.Sprintf("server.%d.requests", room.ServerID)

	log.Printf("[DEBUG] Enviando jogada para o servidor %d (sala %s): %+v\n", room.ServerID, room.ID, card)

	nc.Publish(topic, reqBytes)
}



func StartGameListener(nc *nats.Conn, clientID string, matchChan chan<- MatchInfo, currentUser shared.User) *nats.Subscription {
	clientTopic := fmt.Sprintf("client.%s.inbox", clientID)

	sub, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		var resp shared.Response
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			log.Println("Erro ao decodificar mensagem:", err)
			return
		}

		fmt.Printf("\n[DEBUG] Listener recebeu: %s\n", resp.Action)

		// Só processa mensagem de MATCH
		if resp.Action == "MATCH" {
			var room shared.GameRoom
			if err := json.Unmarshal(resp.Data, &room); err != nil {
				log.Println("Erro ao decodificar sala:", err)
				return
			}

			fmt.Print("\n----------------------------------")
			fmt.Printf("\nSala ID: %s", room.ID)
			fmt.Print("\n----------------------------------")
			fmt.Printf("\nPlayer1: %s", room.Player1.UserName)
			fmt.Printf("\nPlayer2: %s", room.Player2.UserName)
			fmt.Print("\n----------------------------------\n")

			// Determina quem é o adversário
			var opponent shared.User
			if room.Player1.UserId == currentUser.UserId {
				opponent = *room.Player2
			} else {
				opponent = *room.Player1
			}

			// Envia para o canal
			matchChan <- MatchInfo{
				Opponent: opponent,
				Room:     room,
			}
		}
	})

	if err != nil {
		log.Printf("Erro ao iniciar listener: %v", err)
		return nil
	}

	return sub
}

//Processa mensagens de jogo recebidas
func ClientProcessGameMessage(msgData []byte, currentUser shared.User, nc *nats.Conn, room *shared.GameRoom) {
    var msg shared.GameMessage
    if err := json.Unmarshal(msgData, &msg); err != nil {
        return
    }

	fmt.Println("Tipo da mensagem: ", msg.Type)
    switch msg.Type {
    case "PLAY_CARD":
		if len(msg.Data) > 0 {
			var card shared.Card
			if err := json.Unmarshal(msg.Data, &card); err != nil {
				log.Println("Erro ao decodificar carta:", err)
				return
			}
			if msg.From != currentUser.UserId {
				fmt.Printf("\nO oponente jogou: %s (%s)\n", card.Element, card.Type)
			}
		}

		// Sempre verifica se é a vez do usuário
		if msg.Turn == currentUser.UserId {
			fmt.Println("\nSua vez!")
			chosenCard, ok := ChooseCard(currentUser)
			if ok {
				SendCardPlay(nc, room, currentUser.UserId, chosenCard)
			}
		} else {
			fmt.Println("\nAguardando oponente jogar...")
		}
		
    default:
        fmt.Printf("\nTipo desconhecido: '%s'\n", msg.Type)
    }
}
