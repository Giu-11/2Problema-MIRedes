package game

import (
	"fmt"
    "log"
	"sync"
    "bytes"
    "net/http"
	"math/big"
	"crypto/rand"
	"encoding/json"

	"pbl/shared"
	"pbl/server/fsm"
	"pbl/client/utils"

	"github.com/nats-io/nats.go"
)

var (
	GameRooms   = make(map[string]*shared.GameRoom)
	GameRoomsMu sync.RWMutex
)

func CreateRoom(player1, player2 *shared.User, nc *nats.Conn, serverID int) *shared.GameRoom {
    roomID := utils.GenerateRoomID(serverID)
    var turn string
    n, _ := rand.Int(rand.Reader, big.NewInt(2)) //sorteio da vez
    if n.Int64() == 0 {
        turn = player1.UserId
    } else {
        turn = player2.UserId
    }

    room := &shared.GameRoom{
        ID:      roomID,
        Player1: player1,
        Player2: player2,
        Turn:    turn,
        Status:  shared.InProgress,
        ServerID: serverID,
    }

    GameRoomsMu.Lock()
    GameRooms[roomID] = room
    GameRoomsMu.Unlock()

    SendTurnNotification(nc, room) //notifica a vez

    return room
}

//Notificar a vez 
func SendTurnNotification(nc *nats.Conn, room *shared.GameRoom) {
    for _, player := range []*shared.User{room.Player1, room.Player2} {
        msg := shared.GameMessage{
            Type: "PLAY_CARD",
            Turn: room.Turn,
        }

        data, _ := json.Marshal(msg)
        topic := fmt.Sprintf("client.%s.inbox", player.UserId)
        nc.Publish(topic, data)
    }
}

func HandlePlayCardRequest(fsm *fsm.FSM, nc *nats.Conn, serverID int, req shared.Request) {
	var payload struct {
		RoomID string      `json:"roomID"`
		Card   shared.Card `json:"card"`
	}
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		log.Println("Erro ao decodificar payload:", err)
		return
	}

	room, exists := fsm.GlobalRooms[payload.RoomID] // precisa estar exportado
	if !exists {
		log.Printf("Sala não encontrada: %s", payload.RoomID)
	
		return
	}

	if serverID == room.ServerID { // Sou o host
		fsm.ApplyPlayCard(room, req.ClientID, payload.Card)
		notifyClientsCards(nc, room, payload.Card)
	} else { // Não sou host → envio para host via REST
		go func() {
			url := fmt.Sprintf("http://server%d:800%d/notify-play", room.ServerID, room.ServerID)
			jsonData, _ := json.Marshal(struct {
				RoomID   string      `json:"roomID"`
				PlayerID string      `json:"playerID"`
				Card     shared.Card `json:"card"`
			}{
				RoomID:   payload.RoomID,
				PlayerID: req.ClientID,
				Card:     payload.Card,
			})
			if _, err := http.Post(url, "application/json", bytes.NewReader(jsonData)); err != nil {
				log.Printf("Erro ao notificar host: %v", err)
			}
		}()
	}
}

func notifyClientsCards(nc *nats.Conn, room *shared.GameRoom, card shared.Card) {
	for _, player := range []*shared.User{room.Player1, room.Player2} {
		msg := shared.GameMessage{
			Type:     "PLAY_CARD",  
			Turn:     room.Turn,    
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Erro ao serializar mensagem para %s: %v", player.UserId, err)
			continue
		}

		topic := fmt.Sprintf("server.%d.client.%s", player.ServerID, player.UserId)
		if err := nc.Publish(topic, data); err != nil {
			log.Printf("Erro ao enviar mensagem para %s: %v", player.UserId, err)
		}
	}
	log.Printf("[NotifyClients] Mensagem de carta jogada enviada para ambos os players da sala %s", room.ID)
}