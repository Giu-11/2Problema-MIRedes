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
                log.Printf("[%d] - Eleição de %d. Eu (%d) sou maior, enviando OK", s.ID, message.FromID, s.ID)
                response := models.ElectionMessage{
                    Type: "OK",
                    FromID: s.ID,
                    FromURL: s.SelfURL,
                }
                go utils.SendElectionMessage(message.FromURL, response)
                
                s.Mu.Lock()
                if !s.ElectionInProgress {
                    s.Mu.Unlock()
                    log.Printf("[%d] - Iniciando minha própria eleição", s.ID)
                    go StartElection(s)
                } else {
                    s.Mu.Unlock()
                    log.Printf("[%d] - Eleição já em andamento, não inicio nova", s.ID)
                }
            }

        case "OK":
            s.Mu.Lock()
            s.ReceivedOK = true
            s.Mu.Unlock()

        case "LEADER":
            s.Mu.Lock()
            oldLeader := s.Leader
            s.Leader = message.LeaderID
            s.IsLeader = (s.ID == message.LeaderID)
            s.ElectionInProgress = false
            s.ReceivedOK = false
            s.Mu.Unlock()
            log.Printf("[%d] - LÍDER ATUALIZADO de %d para %d (sou líder? %v)", s.ID, oldLeader, s.Leader, s.IsLeader)
            
        default:
            log.Printf("[%d] - Tipo de mensagem desconhecido: %s", s.ID, message.Type)
        }
        
        w.WriteHeader(http.StatusOK)
    }
}

func StartElection(s *models.Server){
    s.Mu.Lock()
    if s.ElectionInProgress{
        //log.Printf("[%d]Eleição já em andamento, ignorando", s.ID)
        s.Mu.Unlock()
        return
    }
    s.ElectionInProgress = true
    s.ReceivedOK = false
    s.Mu.Unlock()

    log.Printf("[%d] - INICIANDO ELEIÇÃO", s.ID)
    log.Printf("[%d] - Meus peers: %+v", s.ID, s.Peers)
    
    higherExist := false

    for _, peer := range s.Peers{
        log.Printf("[%d] - Verificando peer ID=%d, URL=%s", s.ID, peer.ID, peer.URL)
        
        if peer.ID > s.ID{
            //log.Printf("[%d]Peer %d é maior que eu, enviando ELECTION", s.ID, peer.ID)
            message := models.ElectionMessage{
                Type: "ELECTION",
                FromID: s.ID,
                FromURL: s.SelfURL,
            }
            err := utils.SendElectionMessage(peer.URL, message)
            if err != nil {
                log.Printf("[%d] - ERRO ao enviar ELECTION para %d (%s): %v", s.ID, peer.ID, peer.URL, err)
            } else {
                log.Printf("[%d] - ELECTION enviado para %d", s.ID, peer.ID)
            }
            higherExist = true
        } else {
            log.Printf("[%d] - Peer %d é menor ou igual, ignorando", s.ID, peer.ID)
        }
    }
    time.Sleep(3*time.Second)

    s.Mu.Lock()
    receivedOK := s.ReceivedOK

    if !higherExist && !receivedOK {
        s.IsLeader = true
        s.Leader = s.ID
        s.Mu.Unlock()
        

        //Enviadno quem é líder para todos
        for _, peer := range s.Peers{
            message := models.ElectionMessage{
                Type: "LEADER",
                FromID: s.ID,
                LeaderID: s.ID,
                FromURL: s.SelfURL,
            }
            
            err := utils.SendElectionMessage(peer.URL, message)
            if err != nil {
                log.Printf("[%d] - ERRO ao enviar LEADER para %d (%s): %v", s.ID, peer.ID, peer.URL, err)
            } else {
                log.Printf("[%d] - LEADER enviado com sucesso para %d", s.ID, peer.ID)
            }
        }
        
    } else {
        s.Mu.Unlock()
    }
    
    s.Mu.Lock()
    s.ElectionInProgress = false
    s.Mu.Unlock()
    
    log.Printf("[%d] - ELEIÇÃO FINALIZADA", s.ID)
}


//CADASTRO
func HandleRegister() {
}

func HandleLogin(){
    //fazer ainda
}