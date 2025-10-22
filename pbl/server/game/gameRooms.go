package game

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sync"

	"pbl/shared"

	"github.com/nats-io/nats.go"
)

var (
	GameRooms   = make(map[string]*shared.GameRoom)
	GameRoomsMu sync.RWMutex
)

func CreateRoom(player1, player2 *shared.User, nc *nats.Conn, serverID int) *shared.GameRoom {
    roomID := generateRoomID(serverID)
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

func generateRoomID(serverID int) string {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := int(binary.LittleEndian.Uint32(b[:]))
	return fmt.Sprintf("%d-%d", serverID, int(math.Abs(float64(id))))
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

