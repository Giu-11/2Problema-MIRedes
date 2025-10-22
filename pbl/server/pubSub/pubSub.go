package pubSub

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"pbl/server/handlers"
	"pbl/server/models"
	"pbl/shared"

	"github.com/nats-io/nats.go"
)

func StartNats(server *models.Server) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(2 * time.Second),
	}

	nc, err := nats.Connect(os.Getenv("NATS_URL"), opts...)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no NATS: %w", err)
	}

	// Subscribe no tópico de requests do servidor
	topic := fmt.Sprintf("server.%d.requests", server.ID)
	_, err = nc.Subscribe(topic, func(msg *nats.Msg) {
		var req shared.Request
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[%d] Erro ao decodificar request: %v", server.ID, err)
			return
		}

		//Chama o handler correto
		switch req.Action {
		case "CHOOSE_SERVER":
			handlers.HandleChooseServer(server, req, nc, msg)
		case "LOGIN":
			handlers.HandleLogin(server, req, nc, msg)
		case "HEARTBEAT":
			handlers.HandleHeartbeat(server.ID, req, nc, msg)
		case "LOGOUT":
			handlers.HandleLogout(server, req, nc, msg)
		case "JOIN_QUEUE":
			handlers.HandleJoinQueue(server, req, nc, msg)
		case "GAME_MESSAGE":
			log.Println("Entrou no message")
    		handlers.HandleGameMessage(server, req, nc, msg)
		case "OPEN_PACK":
			handlers.HandleDrawCard(server, req, nc, msg)
		case "SEE_CARDS":
			handlers.SeeCardsHandler(server, req, nc, msg)
		case "CHANGE_DECK":
			handlers.HandleChangeDeck(server, req, nc, msg)
		}

	})
	if err != nil {
		return nil, fmt.Errorf("erro ao se inscrever no tópico: %w", err)
	}

	log.Printf("[%d] - Inscrito em %s", server.ID, topic)
	return nc, nil
}
