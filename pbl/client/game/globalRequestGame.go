package game

import (
	"log"
	"fmt"
	"time"
	"encoding/json"

	"pbl/shared"
	"pbl/style"

	"github.com/nats-io/nats.go"
)

func HandleStartGlobalMatchListener(serverID int, nc *nats.Conn, clientID string, matchChan chan<- MatchInfo) *nats.Subscription {
	// Tópico específico do cliente no servidor
	clientTopic := fmt.Sprintf("server.%d.client.%s", serverID, clientID)

	sub, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		var gameMsg shared.GameMessage
		if err := json.Unmarshal(msg.Data, &gameMsg); err != nil {
			log.Println("Erro ao decodificar mensagem de partida global:", err)
			return
		}

		// Apenas processa partidas globais
		if gameMsg.Type == "GLOBAL_MATCH_CREATED" {
			var room shared.GameRoom
			if err := json.Unmarshal(gameMsg.Data, &room); err != nil {
				log.Println("Erro ao decodificar dados da sala:", err)
				return
			}

			// Determina quem é o adversário
			var opponent shared.User
			if room.Player1.UserId == clientID {
				opponent = *room.Player2
			} else {
				opponent = *room.Player1
			}

			// Envia para o canal do cliente
			matchChan <- MatchInfo{
				Opponent: opponent,
				Room:     room,
				IsGlobal: true,
			}

			log.Printf("[Cliente] Nova partida global recebida! Sala: %s, Adversário: %s", room.ID, opponent.UserName)
		}
	})

	if err != nil {
		log.Printf("Erro ao se inscrever no tópico %s: %v", clientTopic, err)
		return nil
	}

	//log.Printf("[Cliente] Inscrito no tópico global: %s", clientTopic)
	return sub
}

func PlayGlobalGame(nc *nats.Conn, room *shared.GameRoom, currentUser shared.User, opponent shared.User) {
	fmt.Println("\nPARTIDA GLOBAL INICIADA")

	isMyTurn := room.Turn == currentUser.UserId
	if isMyTurn {
		fmt.Println("✓ Você começa!")
	} else {
		fmt.Printf("%s começa. Aguarde...\n", opponent.UserName)
	}

	gameOver := false
	alreadyPlayed := false
	gameMsgChan := make(chan shared.GameMessage, 10)

	clientTopic := fmt.Sprintf("server.%d.client.%s", currentUser.ServerID, currentUser.UserId)
	log.Printf("[Cliente] Inscrito no tópico: %s", clientTopic)
	
	sub, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		var gameMsg shared.GameMessage
		if err := json.Unmarshal(msg.Data, &gameMsg); err != nil {
			log.Println("Erro ao decodificar mensagem:", err)
			return
		}
		log.Printf("[Cliente] Mensagem recebida: Type=%s, From=%s", gameMsg.Type, gameMsg.From)
		gameMsgChan <- gameMsg
	})
	if err != nil {
		log.Println("Erro ao criar subscription:", err)
		return
	}
	defer sub.Unsubscribe()


	if isMyTurn {
		card, ok := ChooseCard(currentUser)
		if !ok {
			fmt.Println("Você desistiu da partida.")
			return
		}

		SendCardPlayGlobal(nc, room, currentUser, card)
		fmt.Printf("\nVocê jogou: %s (%s)\n", card.Element, card.Type)
		fmt.Println("Aguardando adversário...")
		alreadyPlayed = true
	}

	for !gameOver {
		select {
		case gameMsg := <-gameMsgChan:
			switch gameMsg.Type {
			case "PLAY_CARD":
				if gameMsg.From == opponent.UserId {
					var card shared.Card
					if len(gameMsg.Data) > 0 {
						if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
							log.Println("Erro ao decodificar carta:", err)
							continue
						}
					}

					//fmt.Printf("\n%s jogou: %s (%s)\n", opponent.UserName, card.Element, card.Type)

					if gameMsg.Turn == currentUser.UserId && !alreadyPlayed {
						fmt.Println("\n✓ Sua vez de jogar!")
						chosenCard, ok := ChooseCard(currentUser)
						if ok {
							SendCardPlayGlobal(nc, room, currentUser, chosenCard)
							fmt.Printf("Você jogou: %s (%s)\n", chosenCard.Element, chosenCard.Type)
							fmt.Println("Aguardando resultado...")
							alreadyPlayed = true
						} else {
							fmt.Println("Você desistiu da partida.")
							return
						}
					} else if alreadyPlayed {
						fmt.Println("Aguardando resultado da rodada...")
					}
				}

			case "ROUND_RESULT":
				fmt.Println("\n--------------------------------")
				fmt.Println("            Resultado           ")
				fmt.Println("--------------------------------")

				if gameMsg.Winner == nil {
					fmt.Println("Empate dos jogadores!")
				} else {
					fmt.Println("Vencedor: ", gameMsg.Winner.UserName)
				}

				alreadyPlayed = false
				gameOver = true

			case "GAME_OVER":
				fmt.Println("\nJogo finalizado!")
				gameOver = true
			}

		case <-time.After(90 * time.Second):
			fmt.Println("\nTimeout: servidor não respondeu.")
			gameOver = true
		}
	}

	fmt.Print("Pressione ENTER para voltar ao menu principal...")
	fmt.Scanln()
	style.Clear()
}

func SendCardPlayGlobal(nc *nats.Conn, room *shared.GameRoom, client shared.User, card shared.Card) {
	dataBytes, _ := json.Marshal(card)

	gameMsg := shared.GameMessage{
		Type:   "PLAY_CARD_GLOBAL",
		From:   client.UserId,
		RoomID: room.ID,
		Data:   dataBytes,
	}

	payload, _ := json.Marshal(gameMsg)

	req := shared.Request{
		ClientID: client.UserId,
		Action:   "GAME_MESSAGE_GLOBAL",
		Payload:  payload,
	}

	reqBytes, _ := json.Marshal(req)
	topic := fmt.Sprintf("server.%d.requests", client.ServerID)

	log.Printf("[DEBUG] Enviando jogada para o servidor %d (sala %s): %+v\n", client.ServerID, room.ID, card)

	nc.Publish(topic, reqBytes)
}
