package game

import (
	"log"
	"fmt"
	"encoding/json"
	
	"pbl/shared"
)
func HandleIncomingGameMessage(msgData []byte) {
	var msg shared.GameMessage
	if err := json.Unmarshal(msgData, &msg); err != nil {
		log.Println("Erro ao decodificar GameMessage:", err)
		return
	}

	if msg.Type == "PLAY_CARD" {
		var play shared.Card
		if err := json.Unmarshal(msg.Data, &play); err != nil {
			log.Println("Erro ao decodificar carta jogada:", err)
			return
		}
		fmt.Printf("\nO oponente jogou: %s (%s)\n", play.Element, play.Type)
	}
}
