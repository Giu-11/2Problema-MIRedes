package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"pbl/server/models"
	"pbl/server/utils"
	"time"
)

func PingHandler(serverID int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var msg models.Message
		if len(body) > 0 {
			json.Unmarshal(body, &msg)
		}

		log.Printf("[%s] Recebi PING de %s", serverID, msg.From)

		resp := models.Message{
			From: serverID,
			Msg:  "PONG",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func ElectionHandler(s *models.Server) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request) {
		var message models.ElectionMessage
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil{
			http.Error(w, "erro ao decodificar", http.StatusBadRequest)
			return 
		}
		switch message.Type{
		case "ELECTION":
			if message.FromID < s.ID{
				log.Printf("[%d]Eleição de %d. Eu sou maior", s.ID, message.FromID)
				response := models.ElectionMessage{
					Type: "OK",
					FromID: s.ID,
					FromURL: s.SelfURL,
				}
				go utils.SendElectionMessage(message.FromURL, response)
				go StartElection(s)
			}else{
				log.Printf("[%d]Recebi eleição de %d mas sou menor", s.ID, message.FromID)
			}

		case "OK":
			s.Mu.Lock()
			s.ElectionInProgress = true
			s.Mu.Unlock()
			log.Printf("[%d]Recebi ok de %d", s.ID, message.FromID)

		case "LEADER":
			s.Mu.Lock()
			s.Leader = message.LeaderID
			s.IsLeader = (s.ID == message.LeaderID)
			s.ElectionInProgress = false
			s.Mu.Unlock()
			log.Printf("[%d] Atualizado líder: %d", s.ID, s.Leader)
		}
		w.WriteHeader(http.StatusOK)
	}

}

func StartElection(s *models.Server){
	s.Mu.Lock()
	if s.ElectionInProgress{
		s.Mu.Unlock()
		return
	}
	s.ElectionInProgress = true
	s.Mu.Unlock()

	log.Printf("[%d]Eleição iniciada", s.ID)
	higherExist := false

	for _, peer := range s.Peers{
		if peer.ID > s.ID{
			message := models.ElectionMessage{
				Type: "ELECTION",
				FromID: s.ID,
				FromURL: s.SelfURL,
			}
			go utils.SendElectionMessage(peer.URL, message)
			higherExist = true
		}
	}
	time.Sleep(3*time.Second)

	s.Mu.Lock()

	if !higherExist{
		s.IsLeader = true
		s.Leader = s.ID
		log.Printf("[%d]Eleito líder", s.ID)

		for _, peer := range s.Peers{
			message := models.ElectionMessage{
				Type: "LEADER",
				FromID: s.ID,
				LeaderID: s.ID,
				FromURL: s.SelfURL,
			}
			go utils.SendElectionMessage(peer.URL, message)
		}
	}
	s.ElectionInProgress = false
	s.Mu.Unlock()
}