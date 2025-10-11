package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"pbl/server/models"
	"pbl/server/utils"
	"pbl/shared"
	"time"

	"github.com/nats-io/nats.go"
)

func PingHandler(serverID int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var msg models.Message
		if len(body) > 0 {
			json.Unmarshal(body, &msg)
		}

		//log.Printf("[%s] Recebi PING de %s", serverID, msg.From)

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
                //log.Printf("[%d] - Eleição de %d. Eu (%d) sou maior, enviando OK", s.ID, message.FromID, s.ID)
                response := models.ElectionMessage{
                    Type: "OK",
                    FromID: s.ID,
                    FromURL: s.SelfURL,
                }
                go utils.SendElectionMessage(message.FromURL, response)
                
                s.Mu.Lock()
                if !s.ElectionInProgress {
                    s.Mu.Unlock()
                    //log.Printf("[%d] - Iniciando minha própria eleição", s.ID)
                    go StartElection(s)
                } else {
                    s.Mu.Unlock()
                    //log.Printf("[%d] - Eleição já em andamento, não inicio nova", s.ID)
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


func StartElection(s *models.Server) {
    s.Mu.Lock()
    if s.ElectionInProgress {
        s.Mu.Unlock()
        return
    }
    s.ElectionInProgress = true
    s.ReceivedOK = false
    s.Mu.Unlock()

    log.Printf("[%d] - INICIANDO ELEIÇÃO", s.ID)

    // Envia mensagem de eleição para todos os peers com ID maior
    for _, peer := range s.Peers {
        if peer.ID > s.ID {
            log.Printf("[%d] - Peer %d é maior, enviando ELECTION", s.ID, peer.ID)
            message := models.ElectionMessage{
                Type:    "ELECTION",
                FromID:  s.ID,
                FromURL: s.SelfURL,
            }
            // Não se preocupe se der erro, significa que o peer está offline
            go utils.SendElectionMessage(peer.URL, message)
        }
    }

    // Espera um tempo para receber respostas "OK"
    time.Sleep(3 * time.Second)

    s.Mu.Lock()
    receivedOK := s.ReceivedOK
    s.Mu.Unlock()

    // Se não recebemos um "OK", significa que ninguém maior está ativo. Nós somos o líder.
    if !receivedOK {
        s.Mu.Lock()
        // Garante que não nos tornamos líder se outro processo já nos atualizou
        if s.Leader == s.ID {
             s.Mu.Unlock()
             return
        }
        s.IsLeader = true
        s.Leader = s.ID
        s.Mu.Unlock()
        
        log.Printf("[%d] - >>> NINGUÉM MAIOR RESPONDEU. EU SOU O NOVO LÍDER! <<<", s.ID)

        // Anuncia para TODOS os peers (maiores e menores) que é o novo líder
        announceMessage := models.ElectionMessage{
            Type:     "LEADER",
            FromID:   s.ID,
            LeaderID: s.ID,
            FromURL:  s.SelfURL,
        }
        for _, peer := range s.Peers {
            go utils.SendElectionMessage(peer.URL, announceMessage)
        }
    } else {
         log.Printf("[%d] - Recebi um OK. Não sou o líder.", s.ID)
    }

    s.Mu.Lock()
    s.ElectionInProgress = false
    s.Mu.Unlock()

    log.Printf("[%d] - ELEIÇÃO FINALIZADA", s.ID)
}

func HandleChooseServer(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg) {
    // Pega o server_id do payload do cliente (mesmo que seja esse servidor)
    var payloadData map[string]int
    if err := json.Unmarshal(request.Payload, &payloadData); err != nil {
        log.Printf("[%d] - Erro ao decodificar payload: %v", server.ID, err)
        return
    }

    chosenServerID := payloadData["server_id"]
    log.Printf("[%d] - Cliente %s escolheu este servidor (ID=%d)", server.ID, request.ClientID, chosenServerID)

    // Resposta para o cliente confirmando que ele escolheu o servidor
    response := shared.Response{
        Status: "success",
        Action: "CHOOSE_SERVER",
        Server: server.ID,
    }
    data, _ := json.Marshal(response)

    if message.Reply != "" {
        nc.Publish(message.Reply, data)
    }
}

//CADASTRO
func HandleRegister(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg) {
    var newUser shared.User
    if err := json.Unmarshal(request.Payload, &newUser); err != nil {
        log.Printf("[%d] Erro no payload de cadastro: %v", server.ID, err)
        return
    }

    log.Printf("[%d] - Cadastro recebido: %+v", server.ID, request.Payload)

    response := shared.Response{
        Status: "success",
        Action: "REGISTER",
        Server: server.ID,
    }
    data, _ := json.Marshal(response)
    nc.Publish(message.Reply, data)
}

func HandleLogin(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg){
    var newUser shared.User
    if err := json.Unmarshal(request.Payload, &newUser); err != nil {
        log.Printf("[%d] Erro no payload de login: %v", server.ID, err)
        return
    }

    log.Printf("[%d] - Login recebido: %+v", server.ID, request.Payload)

    response := shared.Response{
        Status: "success",
        Action: "LOGIN",
        Server: server.ID,
    }

    data, _ := json.Marshal(response)
    nc.Publish(message.Reply, data)
}
