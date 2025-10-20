package game

import (
	"log"
	"fmt"
	"encoding/json"
	
	"pbl/shared"
)
func HandleIncomingGameMessage(msgData []byte) {
    log.Printf("Mensagem bruta recebida: %s", string(msgData))
    
    var msg shared.GameMessage
    if err := json.Unmarshal(msgData, &msg); err != nil {
        log.Println("Erro ao decodificar GameMessage:", err)
        return
    }
    
    // DEBUG: Print da struct completa
    log.Printf("GameMessage decodificado: %+v", msg)
    log.Printf("Tipo de mensagem: '%s' (len=%d)", msg.Type, len(msg.Type))
    
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
    default:
        log.Printf("Mensagem desconhecida recebida - Type: '%s'", msg.Type)
    }
}
