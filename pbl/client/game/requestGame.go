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
	choiceInt, _ := strconv.Atoi(choice) //muda de volra para err depois

	/*if choiceInt == 0 {
    	return shared.Card{}, false //saída voluntária do usuário
	}

	if err != nil || choiceInt < 1 || choiceInt > len(cards) {
		fmt.Println("Escolha inválida!")
		return shared.Card{}, false
	}*/

	selected := cards[choiceInt-1]
	return selected, true
}

func SendCardPlay(nc *nats.Conn, serverID int, fromUserID, opponentUserID string, card shared.Card) {
	dataBytes, err := json.Marshal(card)
	if err != nil {
		log.Println("Erro ao serializar carta:", err)
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
		log.Println("Erro ao enviar carta:", err)
		return
	}

	fmt.Printf("Você jogou: %s (%s)\n", card.Element, card.Type)
}

func StartGameListener(nc *nats.Conn, clientID string) {
	clientTopic := fmt.Sprintf("client.%s.inbox", clientID)

	_, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		HandleIncomingGameMessage(msg.Data)
	})
	if err != nil {
		log.Printf("Erro ao iniciar listener NATS para %s: %v", clientID, err)
		return
	}

	log.Printf("Listener iniciado no tópico %s", clientTopic)
}

//Processa mensagens de jogo recebidas
func HandleIncomingGameMessage(msgData []byte) {
	var msg shared.GameMessage
	if err := json.Unmarshal(msgData, &msg); err != nil {
		log.Println("Erro ao decodificar GameMessage:", err)
		return
	}
	log.Println("Tipo de mensagem: ", msg.Type)
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
		fmt.Printf("\nPartida iniciada! Oponente: %s\n", opponent.UserId)
	default:
		log.Printf("Mensagem desconhecida recebida: %s", msg.Type)
	}
}
