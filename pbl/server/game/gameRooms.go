package game

import (
	"pbl/server/models"
	"pbl/shared"
	"sync"
	"crypto/rand"
	"math"
	"encoding/binary"
)

var (
	GameRooms   = make(map[string]*models.Room)
	GameRoomsMu sync.RWMutex
)

func CreateRoom(player1, player2 shared.User) *models.Room{
	roomID := generateRoomID()
	var turn shared.User
	if rand.Intn(2) == 0 {
		turn = player1
	} else {
		turn = player2
	}

	room := &models.Room{
		ID:        roomID,
		Player1:   player1,
		Player2:   player2,
		Turn:      turn,
		Status:    models.InProgress,
	}

	GameRoomsMu.Lock()
	GameRooms[roomID] = room
	GameRoomsMu.Unlock()

	return room
}


func generateRoomID() int {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := int(binary.LittleEndian.Uint32(b[:]))
	return int(math.Abs(float64(id)))
}
