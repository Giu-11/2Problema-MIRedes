package game

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"pbl/client/models"
	"pbl/shared"

	"github.com/nats-io/nats.go"
)

func JoinQueue(nc *nats.Conn, server models.ServerInfo, user *shared.User, clientTopic string) bool {
	entry := shared.QueueEntry{
		Player:   user,
		ServerID: fmt.Sprintf("%d", server.ID),
		Topic:    clientTopic,
		JoinTime: time.Now(),
	}
	entry.Player.Status = "available"

	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Erro ao serializar QueueEntry: %v", err)
		return false
	}

	req := shared.Request{
		Action:  "JOIN_QUEUE",
		Payload: json.RawMessage(payload),
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		log.Printf("Erro ao serializar Request: %v", err)
		return false
	}

	topic := fmt.Sprintf("server.%d.requests", server.ID)
	msg, err := nc.Request(topic, reqData, 5*time.Second)
	if err != nil {
		if err == nats.ErrTimeout {
			log.Println("Erro: o servidor não respondeu a tempo ao JOIN_QUEUE.")
		} else {
			log.Printf("Erro ao enviar JOIN_QUEUE: %v", err)
		}
		return false
	}

	var resp shared.Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		log.Printf("Erro ao decodificar resposta do servidor: %v", err)
		return false
	}

	if resp.Status == "success" {
		fmt.Println("Entrou na fila com sucesso. Aguardando pareamento...")
		user.Status = "in_queue"
		return true
	}

	fmt.Println("Falha ao entrar na fila:", resp.Error)
	return false
}
