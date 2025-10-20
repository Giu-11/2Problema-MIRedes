package game

import (
	"pbl/shared"
	"sync"
	"crypto/rand"
	"math/big"
	"math"
	"encoding/binary"
	"strconv"
)

var (
	GameRooms   = make(map[string]*shared.GameRoom)
	GameRoomsMu sync.RWMutex
)

func CreateRoom(player1, player2 *shared.User) *shared.GameRoom{
	roomID := generateRoomID()
	var turn *shared.User
	n, _ := rand.Int(rand.Reader, big.NewInt(2)) //sorteia a vez
	if n.Int64() == 0 {
		turn = player1
	} else {
		turn = player2
	}

	room := &shared.GameRoom{
		ID:        roomID,
		Player1:   player1,
		Player2:   player2,
		Turn:      turn,
		Status:    shared.InProgress,
	}

	GameRoomsMu.Lock()
	GameRooms[roomID] = room
	GameRoomsMu.Unlock()

	return room
}

func generateRoomID() string {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := int(binary.LittleEndian.Uint32(b[:]))
	return strconv.Itoa(int(math.Abs(float64(id))))
}

