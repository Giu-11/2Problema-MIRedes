package game

import(
	"log"
	"fmt"
	"strconv"
	"encoding/json"

	"pbl/client/utils"
	"pbl/shared"

	"github.com/nats-io/nats.go"
)
//preciso da parte das cartas primeiro pra continuar implementando isso aqui
func ChooseCard(user shared.User) (shared.Card, bool) {
	
	utils.ListCardsDeck(&user)
	cards := user.Cards

	fmt.Print("Insira a carta desejada (0 para sair): ")

	choice := utils.ReadLineSafe()
	choiceInt, err := strconv.Atoi(choice) 

	if choiceInt == 0 {
    	return shared.Card{}, false //saída voluntária do usuário
	}

	if err != nil || choiceInt < 1 || choiceInt > len(cards) {
		fmt.Println("Escolha inválida!")
		return shared.Card{}, false
	}

	selected := cards[choiceInt-1]
	return selected, true
}

func SendCardPlay(nc *nats.Conn, serverID int, fromUserID, opponentUserID string, card shared.Card) {
	dataBytes, err := json.Marshal(card)
	if err != nil {
		fmt.Println("Erro ao serializar carta:", err)
		return
	}

	msg := shared.GameMessage{
		Type:     "PLAY_CARD",
		Data:     dataBytes,
		From:     fromUserID,
		ServerID: serverID,
	}

	bytes, _ := json.Marshal(msg)

	topic := fmt.Sprintf("client.%s.inbox", opponentUserID)
	if err := nc.Publish(topic, bytes); err != nil {
		fmt.Println("Erro ao enviar carta:", err)
		return
	}

	fmt.Printf("\nVocê jogou: %s (%s)\n", card.Element, card.Type)
}

func StartGameListener(nc *nats.Conn, clientID string, matchChan chan<- shared.User) {
	clientTopic := fmt.Sprintf("client.%s.inbox", clientID)

	_, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		
		//Tenta primeiro como Response
		var resp shared.Response
		if err := json.Unmarshal(msg.Data, &resp); err == nil && resp.Action == "MATCH" {
			
			//Decodifica a sala completa
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
			fmt.Print("\n----------------------------------")
			
			//Identifica quem é o oponente
			var opponent *shared.User
			if room.Player1.UserId == clientID {
				opponent = room.Player2
			} else {
				opponent = room.Player1
			}
			
			fmt.Printf("\nPartida iniciada! Oponente: %s\n", opponent.UserName)
			
			//Envia para o canal se existir
			if matchChan != nil {
				select {
				case matchChan <- *opponent:
					//fmt.Println("Adversário enviado para canal")
				default:
					fmt.Println("Canal de match cheio ou fechado")
				}
			}
			return
		}
		
		//Se não for MATCH, processa como mensagem de jogo
		HandleIncomingGameMessage(msg.Data)
	})
	
	if err != nil {
		fmt.Printf("\nErro ao iniciar listener: %v", err)
		return
	}

	//fmt.Printf("\nListener iniciado no tópico %s", clientTopic)
}

//Processa mensagens de jogo recebidas
func HandleIncomingGameMessage(msgData []byte) {
	var msg shared.GameMessage
	if err := json.Unmarshal(msgData, &msg); err != nil {
		return
	}
	
	fmt.Printf("\nGameMessage recebida - Type: '%s'", msg.Type)
	
	switch msg.Type {
	case "PLAY_CARD":
		var card shared.Card
		if err := json.Unmarshal(msg.Data, &card); err != nil {
			log.Println("\nErro ao decodificar carta:", err)
			return
		}
		fmt.Printf("\nO oponente jogou: %s (%s)\n", card.Element, card.Type)
		
	default:
		fmt.Printf("\nTipo desconhecido: '%s'", msg.Type)
	}
}
