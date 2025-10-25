package game

import(
	"fmt"
	"log"
	"bytes"
    "math/big"
	"net/http"
	"crypto/rand"
	"encoding/json"

	"pbl/shared"

)


func NotifyGlobalMatch(room *shared.GameRoom) {
	//Sorteia um dos servidores de jogo
	hostIndex, _ := rand.Int(rand.Reader, big.NewInt(2))
	hostNum := hostIndex.Int64() + 1 

	url := fmt.Sprintf("http://gameserver%d:8080/start-game", hostNum)

	jsonData, err := json.Marshal(room)
	if err != nil {
		log.Printf("[NotifyGlobalMatchHTTP] erro ao serializar room: %v", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[NotifyGlobalMatchHTTP] erro ao notificar servidor %d: %v", hostNum, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[NotifyGlobalMatchHTTP] Sala %s enviada para servidor %d (%s vs %s)",
		room.ID, hostNum, room.Player1.UserName, room.Player2.UserName)
}

