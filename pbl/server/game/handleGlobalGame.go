package game

import(
	"fmt"
	"log"
	"bytes"

	"net/http"
	"encoding/json"

	"pbl/shared"

)

// Notifica o servidor host sobre uma nova partida global
func NotifyGlobalMatch(room *shared.GameRoom) {
	hostServerID := room.ServerID // já definido no FSM quando a sala foi criada
	url := fmt.Sprintf("http://gameserver%d:8080/start-game", hostServerID)

	// Serializa a room em JSON
	jsonData, err := json.Marshal(room)
	if err != nil {
		log.Printf("[NotifyGlobalMatch] Erro ao serializar room: %v", err)
		return
	}

	// Envia a requisição HTTP POST para o servidor host
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[NotifyGlobalMatch] Erro ao notificar servidor %d: %v", hostServerID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NotifyGlobalMatch] Servidor %d respondeu com status: %d", hostServerID, resp.StatusCode)
		return
	}

	log.Printf("[NotifyGlobalMatch] Sala %s enviada para servidor %d (%s vs %s)",
		room.ID, hostServerID, room.Player1.UserName, room.Player2.UserName)
}


