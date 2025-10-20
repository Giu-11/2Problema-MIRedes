package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pbl/server/models"
	sharedRaft "pbl/server/shared"
	"strconv"
	"sync"
	"time"

	//sharedRaft "pbl/server/shared"
	"pbl/shared"

	//"github.com/hashicorp/raft"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
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

// Helper para converter qualquer struct em json.RawMessage
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

func HandleDrawCard(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg) {
	if server.Raft.State() == raft.Leader {
		// Se já somos o líder, processamos, salvamos localmente e respondemos.
		result, err := processDrawCardRequest(server, request.ClientID)
		if err != nil {
			respondWithError(nc, message, err.Error())
			return
		}
		saveCardToLocalUser(server, request.ClientID, result)
		respondWithSuccess(nc, message, result)
		return
	}

	// Se não somos o líder, descobrimos quem é e encaminhamos via HTTP REST.
	leaderAddr := server.Raft.Leader()
	if leaderAddr == "" {
		respondWithError(nc, message, "Líder não disponível no momento, tente novamente.")
		return
	}

	log.Printf("[%d] Não sou o líder. Encaminhando 'Pegar Carta' para o líder em %s", server.ID, leaderAddr)

	// O formato da mensagem para o líder é um JSON com o clientID.
	payload := map[string]string{"clientID": request.ClientID}
	jsonPayload, _ := json.Marshal(payload)

	// Faz uma requisição HTTP POST para o endpoint REST do líder.
	resp, err := http.Post(fmt.Sprintf("http://%s/leader/draw-card", leaderAddr), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		respondWithError(nc, message, fmt.Sprintf("Falha ao se comunicar com o líder: %v", err))
		return
	}
	defer resp.Body.Close()

	// Repassa a resposta do líder diretamente para o cliente via NATS.
	var leaderResponse shared.Response
	if err := json.NewDecoder(resp.Body).Decode(&leaderResponse); err != nil {
		respondWithError(nc, message, "Resposta inválida do líder.")
		return
	}

	if leaderResponse.Status == "success" {
		var drawnData shared.CardDrawnData
		if err := json.Unmarshal(leaderResponse.Data, &drawnData); err != nil {
			respondWithError(nc, message, "Dados da carta inválidos na resposta do líder.")
			return
		}
		saveCardToLocalUser(server, request.ClientID, drawnData.Card)
	}

	finalResponseBytes, _ := json.Marshal(leaderResponse)
	nc.Publish(message.Reply, finalResponseBytes)
}

func LeaderDrawCardHandler(server *models.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if server.Raft.State() != raft.Leader {
			http.Error(w, "Eu não sou o líder", http.StatusServiceUnavailable)
			return
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Payload da requisição inválido", http.StatusBadRequest)
			return
		}
		clientID := payload["clientID"]

		result, err := processDrawCardRequest(server, clientID)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			response := shared.Response{Status: "error", Error: err.Error(), Server: server.ID}
			json.NewEncoder(w).Encode(response)
			return
		}

		// O líder também salva a carta se o jogador estiver conectado a ele.
		saveCardToLocalUser(server, clientID, result)

		// Prepara a resposta para o servidor que encaminhou
		responseData := shared.CardDrawnData{Card: result, RequestID: "n/a for forwarded req"}
		responseBytes, _ := json.Marshal(responseData)
		response := shared.Response{Status: "success", Action: "CARD_DRAWN", Data: responseBytes, Server: server.ID}
		json.NewEncoder(w).Encode(response)
	}
}

// saveCardToLocalUser adiciona a carta ao inventário do usuário no servidor local.
func saveCardToLocalUser(server *models.Server, clientID string, card shared.Card) {
	server.Mu.Lock()
	defer server.Mu.Unlock()

	if user, ok := server.Users[clientID]; ok {
		user.Cards = append(user.Cards, card)
		server.Users[clientID] = user
		log.Printf("[%d] Carta '%s' adicionada ao inventário local do cliente %s.", server.ID, card.Type, clientID)
	}
}

func processDrawCardRequest(server *models.Server, clientID string) (shared.Card, error) {
	requestID := uuid.New().String()

	payload := sharedRaft.DrawCardPayload{PlayerID: clientID, RequestID: requestID}
	payloadBytes, _ := json.Marshal(payload)
	cmd := sharedRaft.Command{Type: sharedRaft.CommandOpenPack, Data: payloadBytes}
	cmdBytes, _ := json.Marshal(cmd)

	future := server.Raft.Apply(cmdBytes, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		log.Printf("[%d] Erro ao aplicar comando Raft 'DrawCard': %v", server.ID, err)
		return shared.Card{}, fmt.Errorf("erro interno ao processar a jogada")
	}

	responseValue := future.Response()
	if strValue, ok := responseValue.(string); ok && strValue == "STOCK_EMPTY" {
		return shared.Card{}, fmt.Errorf("o estoque de cartas acabou")
	}

	drawnCard, ok := responseValue.(shared.Card)
	if !ok {
		return shared.Card{}, fmt.Errorf("erro inesperado no tipo de resposta do Raft (esperava shared.Card)")
	}

	log.Printf("[%d] Carta '%s' reservada para o cliente %s (RequestID: %s).", server.ID, drawnCard.Type, clientID, requestID)
	go claimCard(server, requestID)

	return drawnCard, nil
}

// finaliza a transação, removendo a carta da área de pendentes
func claimCard(server *models.Server, requestID string) {
	log.Printf("[%d] Reivindicando carta para o RequestID: %s", server.ID, requestID)

	payload := sharedRaft.ClaimCardPayload{RequestID: requestID}
	payloadBytes, _ := json.Marshal(payload)

	cmd := sharedRaft.Command{
		Type: sharedRaft.CommandClaimCard,
		Data: payloadBytes,
	}
	cmdBytes, _ := json.Marshal(cmd)

	future := server.Raft.Apply(cmdBytes, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		log.Printf("[%d] ERRO CRÍTICO: Falha ao reivindicar a carta para o RequestID %s: %v", server.ID, requestID, err)
	}
}

func SeeCardsHandler(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg){
	server.Mu.Lock()
	cards := shared.Cards {
		Cards : server.Users[request.ClientID].Cards,
	}
	server.Mu.Unlock()
	resp := shared.Response{
		Status: "success",
		Action: "SEE_CARDS",
		Data: mustMarshal(cards),
		Server: server.ID,
	}
	data, _ := json.Marshal(resp)
	nc.Publish(message.Reply, data)
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

// Colocar cliente na fila --> tem dois tipos de fila, a do servidor que ele tá e a "global"
func HandleJoinQueue(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
	var entry shared.QueueEntry
	if err := json.Unmarshal(request.Payload, &entry); err != nil {
		log.Printf("Erro ao desserializar payload do JOIN_QUEUE: %v", err)
		resp := shared.Response{
			Status: "error",
			Error:  "Payload inválido",
		}
		data, _ := json.Marshal(resp)
		nc.Publish(msg.Reply, data)
		return
	}

	// Adiciona na fila local
	server.Matchmaking.Mutex.Lock()
	server.Matchmaking.LocalQueue = append(server.Matchmaking.LocalQueue, entry)
	server.Matchmaking.Mutex.Unlock()

	log.Printf("Cliente %s entrou na fila local do servidor %d", entry.Player.UserName, server.ID)

	//Se for líder, adiciona também na fila global
	if server.Matchmaking.IsLeader {
		server.Matchmaking.Mutex.Lock()
		server.Matchmaking.GlobalQueue = append(server.Matchmaking.GlobalQueue, entry)
		server.Matchmaking.Mutex.Unlock()
		log.Printf("Cliente %s adicionado na fila global (servidor líder)", entry.Player.UserName)
	}

	//Responde para o cliente
	msgData, _ := json.Marshal("Cliente adicionado à fila com sucesso")
	resp := shared.Response{
		Status: "success",
		Data:   msgData,
		Server: server.ID,
	}
	data, _ := json.Marshal(resp)
	nc.Publish(msg.Reply, data)
}

// Função auxiliar para enviar respostas de sucesso
func respondWithSuccess(nc *nats.Conn, msg *nats.Msg, card shared.Card) {
	responseData := shared.CardDrawnData{Card: card, RequestID: "client-facing-id"}
	responseBytes, _ := json.Marshal(responseData)
	response := shared.Response{Status: "success", Action: "CARD_DRAWN", Data: responseBytes}
	finalBytes, _ := json.Marshal(response)
	nc.Publish(msg.Reply, finalBytes)
}

// Função auxiliar para enviar respostas de erro
func respondWithError(nc *nats.Conn, msg *nats.Msg, errorMsg string) {
	response := shared.Response{Status: "error", Error: errorMsg}
	data, _ := json.Marshal(response)
	nc.Publish(msg.Reply, data)
}

//CADASTRO
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
