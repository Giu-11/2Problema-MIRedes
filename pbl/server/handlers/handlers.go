package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"pbl/server/models"
)

func PingHandler(serverID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var msg models.Message
		if len(body) > 0 {
			json.Unmarshal(body, &msg)
		}

		log.Printf("[%s] Recebi PING de %s", serverID, msg.From)

		resp := models.Message{
			From: "server:" + serverID,
			Msg:  "PONG",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
