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

func SendCardPlay(nc *nats.Conn, serverID int, fromUserID, opponentUserID string, card shared.Card){
	data := shared.Card{
		Element: card.Element,
    	Type:    card.Type,
    	Id:  card.Id,
	}
	dataBytes, err := json.Marshal(data)
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
	nc.Publish(topic, bytes)

	fmt.Printf("Você jogou: %s (%s)\n", card.Element, card.Type)
}

