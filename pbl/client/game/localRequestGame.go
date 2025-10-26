package game

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"pbl/client/utils"
	"pbl/shared"
	"pbl/style"

	"github.com/nats-io/nats.go"
)

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

func SendCardPlayLocal(nc *nats.Conn, room *shared.GameRoom, fromUserID string, card shared.Card) {
	log.Println("\033[31mpucou o sendcard paly local\033[0m")
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

		//fmt.Printf("\n[DEBUG] Listener recebeu: %s\n", resp.Action)

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


func PlayLocalGame(nc *nats.Conn, room *shared.GameRoom, currentUser shared.User, opponent shared.User) {
	fmt.Println("\nPARTIDA INICIADA")

	isMyTurn := room.Turn == currentUser.UserId
	if isMyTurn {
		fmt.Println("✓ Você começa!")
	} else {
		fmt.Printf("%s começa. Aguarde...\n", opponent.UserName)
	}

	gameOver := false
	alreadyPlayed := false // faz o controle das jogadas
	gameMsgChan := make(chan shared.GameMessage, 10)

	clientTopic := fmt.Sprintf("client.%s.inbox", currentUser.UserId)
	sub, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		var gameMsg shared.GameMessage
		if err := json.Unmarshal(msg.Data, &gameMsg); err != nil {
			log.Println("Erro ao decodificar mensagem:", err)
			return
		}
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
		SendCardPlayLocal(nc, room, currentUser.UserId, card)
		fmt.Printf("\nVocê jogou: %s (%s)\n", card.Element, card.Type)
		fmt.Println("Aguardando adversário...")
		alreadyPlayed = true //já jogou
	}

	for !gameOver {
		select {
		case gameMsg := <-gameMsgChan:
			switch gameMsg.Type {
			case "PLAY_CARD":
				//Mensagem do adversário jogando
				if gameMsg.From == opponent.UserId {
					var card shared.Card
					if len(gameMsg.Data) > 0 {
						if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
							log.Println("Erro ao decodificar carta:", err)
							continue
						}
					}

					fmt.Printf("\n%s jogou: %s (%s)\n", opponent.UserName, card.Element, card.Type)

					if !alreadyPlayed {
						room.Turn = currentUser.UserId
						fmt.Println("\nSua vez!")
						
						chosenCard, ok := ChooseCard(currentUser)
						if ok {
							SendCardPlayLocal(nc, room, currentUser.UserId, chosenCard)
							fmt.Printf("Você jogou: %s (%s)\n", chosenCard.Element, chosenCard.Type)
							fmt.Println("Aguardando resultado...")
							alreadyPlayed = true //cliente jogou
						} else {
							fmt.Println("Você desistiu da partida.")
							return
						}
					} else {
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
