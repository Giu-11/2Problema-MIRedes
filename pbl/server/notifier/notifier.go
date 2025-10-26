package notifier

import (
	"encoding/json"
	"fmt"
	"log"

	"pbl/shared"

	"github.com/nats-io/nats.go"
)

// NotifyGlobalMatch envia uma mensagem de que uma partida global foi criada
// para os dois jogadores envolvidos na sala.
func NotifyGlobalMatch(nc *nats.Conn, room *shared.GameRoom) {
	if nc == nil {
		log.Println("[Notifier] Conexão NATS nula — não foi possível notificar jogadores.")
		return
	}

	// Serializa a sala
	roomBytes, err := json.Marshal(room)
	if err != nil {
		log.Printf("[Notifier] Erro ao serializar room: %v", err)
		return
	}

	// Cria a mensagem padronizada
	msgBytes, err := json.Marshal(shared.GameMessage{
		Type: "GLOBAL_MATCH_CREATED",
		Data: json.RawMessage(roomBytes),
	})
	if err != nil {
		log.Printf("[Notifier] Erro ao serializar GameMessage: %v", err)
		return
	}

	// Publica para Player1
	topic1 := fmt.Sprintf("server.%d.client.%s", room.Server1ID, room.Player1.UserId)
	if err := nc.Publish(topic1, msgBytes); err != nil {
		log.Printf("[Notifier] Erro ao publicar para %s: %v", topic1, err)
	}

	// Publica para Player2
	topic2 := fmt.Sprintf("server.%d.client.%s", room.Server2ID, room.Player2.UserId)
	if err := nc.Publish(topic2, msgBytes); err != nil {
		log.Printf("[Notifier] Erro ao publicar para %s: %v", topic2, err)
	}

	log.Printf("[Notifier] GLOBAL_MATCH_CREATED enviado para %s e %s",
		room.Player1.UserName, room.Player2.UserName)
}
