package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"pbl/server/models"
	"strconv"
	"sync"
	"time"

	//sharedRaft "pbl/server/shared"
	"pbl/shared"
	"pbl/server/game"

	//"github.com/hashicorp/raft"
	"github.com/nats-io/nats.go"
)

type ClientInfo struct {
	ClientID string
	LastSeen time.Time
}

var (
	activeClients = make(map[string]*ClientInfo)
	mu            = sync.Mutex{}
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
/*
func HandleLogin(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg) {
	//TODO: lidar com clientes já logados(não logar quem já logou)
	var userCredentials shared.User
	if err := json.Unmarshal(request.Payload, &userCredentials); err != nil {
		log.Printf("[%d] - Erro no payload de login: %v", server.ID, err)
		return
	}

	//Bloqueia o mapa de usuários para evitar problemas de concorrência
	server.Mu.Lock()
	defer server.Mu.Unlock()

	// Verifica se o usuário já existe no mapa LOCAL deste servidor
	existingUser, exists := server.Users[userCredentials.UserName]

	var response shared.Response

	if exists {
		// Usuário existe, verifica a senha
		if existingUser.Password == userCredentials.Password {
			log.Printf("[%d] - Usuário '%s' logado com sucesso.", server.ID, userCredentials.UserName)
			response = shared.Response{
				Status: "success",
				Action: "LOGIN_SUCCESS",
				Server: server.ID,
			}
		} else {
			log.Printf("[%d] - Tentativa de login falhou para '%s': senha incorreta.", server.ID, userCredentials.UserName)
			response = shared.Response{
				Status: "error",
				Action: "LOGIN_FAIL",
				Error:  "Senha incorreta.",
				Server: server.ID,
			}
		}
	} else {
		log.Printf("[%d] - Usuário '%s' não encontrado. Criando novo usuário local.", server.ID, userCredentials.UserName)
		server.Users[userCredentials.UserName] = userCredentials
		response = shared.Response{
			Status: "success",
			Action: "LOGIN_SUCCESS",
			Server: server.ID,
		}
	}

	data, _ := json.Marshal(response)
	nc.Publish(message.Reply, data)
}
*/

func HandleLogin(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
    var user shared.User
    if err := json.Unmarshal(request.Payload, &user); err != nil {
        log.Printf("[%d] - Erro ao desserializar login: %v", server.ID, err)
        resp := shared.Response{
            Status: "error",
            Action: "LOGIN_FAIL",
            Error:  "payload inválido",
            Server: server.ID,
        }
        data, _ := json.Marshal(resp)
        nc.Publish(msg.Reply, data)
        return
    }

    //Bloqueia o mapa de usuários para evitar condições de corrida
    server.Mu.Lock()
    server.Users[request.ClientID] = user
    server.Mu.Unlock()

    log.Printf("[%d] - Usuário '%s' conectado com ClientID '%s'", server.ID, user.UserName, request.ClientID)

    //Retorna os dados completos do usuário para o cliente
    resp := shared.Response{
        Status: "success",
        Action: "LOGIN_SUCCESS",
        Data:   mustMarshal(user), //converte struct User em JSON
        Server: server.ID,
    }
    data, _ := json.Marshal(resp)
    nc.Publish(msg.Reply, data)
}

//Helper para converter qualquer struct em json.RawMessage
func mustMarshal(v interface{}) json.RawMessage {
    b, _ := json.Marshal(v)
    return json.RawMessage(b)
}

func HandleLogout(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
    server.Mu.Lock()
    user, exists := server.Users[request.ClientID]
    if exists {
        log.Printf("[%d] - Cliente '%s' desconectado (ClientID: %s)", server.ID, user.UserName, request.ClientID)
        delete(server.Users, request.ClientID)
    } else {
        log.Printf("[%d] - Cliente com ClientID '%s' desconectado (usuário não encontrado)", server.ID, request.ClientID)
    }
    server.Mu.Unlock()

    mu.Lock()
    delete(activeClients, request.ClientID)
    mu.Unlock()

    DisconnectClient(server, request.ClientID)

    // Resposta para o cliente
    resp := shared.Response{
        Status: "success",
        Action: "LOGOUT_SUCCESS",
        Server: server.ID,
    }
    data, _ := json.Marshal(resp)
    nc.Publish(msg.Reply, data)
}

// Heatbeat para o clinete
func StartHeartbeatMonitor(server *models.Server, nc *nats.Conn) {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			now := time.Now()

			mu.Lock()
			for id, c := range activeClients {
				if now.Sub(c.LastSeen) > 15*time.Second {
					log.Printf("Cliente '%s' inativo. Removendo...", id)
					delete(activeClients, id)

					//remove do mapa de usuários do servidor
					server.Mu.Lock()
					delete(server.Users, id)
					server.Mu.Unlock()

					response := shared.Response{
						Status: "success",
						Action: "LOGOUT_SUCCESS",
						Server: server.ID,
					}
					data, _ := json.Marshal(response)
					nc.Publish("server."+strconv.Itoa(server.ID)+".requests", data)
				}
			}
			mu.Unlock()
		}
	}()
}

// Função que trata a desconexão de forma genérica
func DisconnectClient(server *models.Server, clientID string) {
	server.Mu.Lock()
	delete(server.Users, clientID)
	server.Mu.Unlock()

	mu.Lock()
	delete(activeClients, clientID)
	mu.Unlock()

	log.Printf("Cliente '%s' caiu ou ficou inativo. Removido do servidor.", clientID)
}

// Handler para processar os heartbeats recebidos via NATS
func HandleHeartbeat(serverID int, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
	clientID := request.ClientID
	mu.Lock()
	activeClients[clientID] = &ClientInfo{
		ClientID: clientID,
		LastSeen: time.Now(),
	}
	mu.Unlock()

	//log.Printf("[%d] - Heartbeat recebido de %s", serverID, clientID)

}

func HandleGameMessage(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
    game.HandleIncomingGameMessage(request.Payload)
}


//CADASTRO --> Jogado fora por falta de tempo
/*func HandleRegister(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg) {
    // 1. Verifica se este nó é o líder. Só o líder deve aceitar escritas.
    if server.Raft.State() != raft.Leader {
        // Redireciona para o líder ou retorna um erro
        // Por simplicidade, vamos retornar um erro agora.
        response := shared.Response{Status: "error", Error: "Not the leader. Try again later."}
        data, _ := json.Marshal(response)
        nc.Publish(message.Reply, data)
        return
    }

    // 2. Monta o comando para o log do Raft
    cmd := sharedRaft.Command{
        Type: sharedRaft.CommandRegisterUser,
        Data: request.Payload, //O payload já é o JSON do usuário
    }
    cmdBytes, err := json.Marshal(cmd)
    if err != nil {
        return
    }

    // 3. Aplica o comando ao log do Raft. Isso vai bloquear até ser replicado.
    future := server.Raft.Apply(cmdBytes, 500*time.Millisecond)
    if err := future.Error(); err != nil {
        log.Printf("[%d] Erro ao aplicar comando Raft: %v", server.ID, err)
        // ... trata o erro ...
        return
    }

    // 4. O comando foi replicado com sucesso! Responde ao cliente.
    log.Printf("[%d] - Cadastro replicado com sucesso via Raft.", server.ID)
    response := shared.Response{
        Status: "success",
        Action: "REGISTER",
        Server: server.ID,
    }
    data, _ := json.Marshal(response)
    nc.Publish(message.Reply, data)
}*/
