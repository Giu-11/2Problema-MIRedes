package handlers

import (
	//"io"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"strings"

	"pbl/server/game"
	"pbl/server/models"

	sharedRaft "pbl/server/shared"
	"pbl/server/utils"
	"pbl/shared"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	"github.com/nats-io/nats.go"
)

const (
	Active ClientState = iota
	WaitingReconnection
)

type ClientState int
type ClientInfo struct {
	ClientID string
	LastSeen time.Time
	State ClientState
}

const (
	heartbeatInterval = 10 * time.Second
	disconnectTimeout = 30 * time.Second
)

var (
	activeClients = make(map[string]*ClientInfo)
	mu            = sync.Mutex{}
)

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

    server.Mu.Lock()
    defer server.Mu.Unlock()

    //Verifica se já existe um usuário com o mesmo nome online
    for _, existingUser := range server.Users {
        if existingUser.UserName == user.UserName {
            log.Printf("[%d] - Tentativa de login duplicado para '%s'", server.ID, user.UserName)
            resp := shared.Response{
                Status: "error",
                Action: "LOGIN_FAIL",
                Error:  "Usuário já está logado em outro cliente.",
                Server: server.ID,
            }
            data, _ := json.Marshal(resp)
            nc.Publish(msg.Reply, data)
            return
        }
    }

    //Insere cartas padrão 
    user.Cards = []shared.Card{
        {Id: "1", Element: "AGUA", Type: "NORMAL"},
        {Id: "2", Element: "TERRA", Type: "NORMAL"},
        {Id: "3", Element: "FOGO", Type: "NORMAL"},
        {Id: "4", Element: "AR", Type: "NORMAL"},
        {Id: "5", Element: "MATO", Type: "NORMAL"},
    }

    user.Deck = []shared.Card{
        {Id: "1", Element: "AGUA", Type: "NORMAL"},
        {Id: "2", Element: "TERRA", Type: "NORMAL"},
        {Id: "3", Element: "FOGO", Type: "NORMAL"},
        {Id: "4", Element: "AR", Type: "NORMAL"},
    }

	user.ServerID = server.ID
    //Armazena o usuário logado
    server.Users[request.ClientID] = user
    log.Printf("[%d] - Usuário '%s' conectado com ClientID '%s'", server.ID, user.UserName, request.ClientID)

    resp := shared.Response{
        Status: "success",
        Action: "LOGIN_SUCCESS",
        Data:   utils.MustMarshal(user),
        Server: server.ID,
    }
    data, _ := json.Marshal(resp)
    nc.Publish(msg.Reply, data)
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

//Para fora do compose
/*func HandleDrawCard(server *models.Server, request shared.Request, nc *nats.Conn, message *nats.Msg) {
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
	log.Printf("\033[31mEDENREÇO DO RADT:%s\033[0m", leaderAddr)
	
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
}*/

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
	leaderAddr := string(server.Raft.Leader())
	if leaderAddr == "" {
		respondWithError(nc, message, "Líder não disponível no momento, tente novamente.")
		return
	}

	log.Printf("[%d] Não sou o líder. Endereço do líder retornado pelo Raft: %s", server.ID, leaderAddr)
	
	//Como no raft tá vindo 0.0.0.0 no inicio, aqui eu faço uma "conversão" --> para funcionar no compose
	parts := strings.Split(leaderAddr, ":")
	host := parts[0]
	port := parts[1]

	// Substitui 0.0.0.0 pelo container correto do líder
	if host == "0.0.0.0" {
		switch port {
		case "8001":
			host = "server1"
		case "8002":
			host = "server2"
		case "8003":
			host = "server3"
		}
	}

	leaderURL := fmt.Sprintf("http://%s:%s/leader/draw-card", host, port)

	// Cria payload para o líder
	payload := map[string]string{"clientID": request.ClientID}
	jsonPayload, _ := json.Marshal(payload)

	// Faz a requisição HTTP POST para o líder
	resp, err := http.Post(leaderURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		respondWithError(nc, message, fmt.Sprintf("Falha ao se comunicar com o líder: %v", err))
		return
	}
	defer resp.Body.Close()

	// Repassa a resposta do líder para o cliente via NATS
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
		Data: utils.MustMarshal(cards),
		Server: server.ID,
	}
	data, _ := json.Marshal(resp)
	nc.Publish(message.Reply, data)
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


func HandleGameMessage(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
    var gameMsg shared.GameMessage
    if err := json.Unmarshal(request.Payload, &gameMsg); err != nil {
        log.Println("Erro ao decodificar GameMessage:", err)
        return
    }

    //Pega a sala do jogador
    roomID := gameMsg.RoomID
    game.GameRoomsMu.Lock()
    room, exists := game.GameRooms[roomID]
    game.GameRoomsMu.Unlock()
    if !exists {
        log.Println("Sala não encontrada:", roomID)
        return
    }

    //Inicializa mapa de cartas
    if room.PlayersCards == nil {
        room.PlayersCards = make(map[string]shared.Card)
    }

    //Decodifica a carta jogada
    var card shared.Card
    if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
        log.Println("Erro ao decodificar carta:", err)
        return
    }
    room.PlayersCards[gameMsg.From] = card
	log.Printf("\033[31mCARTA RECEBIDA: %+v\033[0m", card)
    fmt.Println("Carta que chegou: ", card)

    //Determina quem será o próximo
    var nextTurn string
    if gameMsg.From == room.Player1.UserId {
        nextTurn = room.Player2.UserId
    } else {
        nextTurn = room.Player1.UserId
    }
    room.Turn = nextTurn

    //Envia mensagem para ambos os jogadores
    turnMsg := shared.GameMessage{
        Type: "PLAY_CARD",
        From: gameMsg.From,  
        Data: gameMsg.Data,  
        Turn: nextTurn,      
    }
    dataTurn, _ := json.Marshal(turnMsg)

    /*for _, player := range []*shared.User{room.Player1, room.Player2} {
        nc.Publish(fmt.Sprintf("client.%s.inbox", player.UserId), dataTurn)
    }*/
    opponentID := room.Player1.UserId
    if gameMsg.From == room.Player1.UserId{
        opponentID = room.Player2.UserId
    }

    nc.Publish(fmt.Sprintf("client.%s.inbox", opponentID), dataTurn)

    //Se ambos jogaram, calcula resultado
    if len(room.PlayersCards) == 2 {
        cardP1 := room.PlayersCards[room.Player1.UserId]
        cardP2 := room.PlayersCards[room.Player2.UserId]
        //fmt.Println("Carta p1: ", cardP1)
        //fmt.Println("Carta p2: ", cardP2)
        resultP1 := game.CheckWinner(cardP1, cardP2)
        NotifyResult(nc, room, resultP1)

        // Limpa cartas para a próxima rodada
        room.PlayersCards = make(map[string]shared.Card)
        return
    }
}

func HandleChangeDeck(server *models.Server, request shared.Request, nc *nats.Conn, msg *nats.Msg){
	var deck []shared.Card
	if err := json.Unmarshal(request.Payload, &deck); err != nil{
		log.Printf("[%d] - Erro ao desserializar o deck: %v", server.ID, err)
		resp := shared.Response{
            Status: "error",
            Action: "CHANGE_DECK_FAIL",
            Error:  "payload inválido",
            Server: server.ID,
        }
		data, _ := json.Marshal(resp)
        nc.Publish(msg.Reply, data)
        return
	}

	server.Mu.Lock()
	defer server.Mu.Unlock()

	user := server.Users[request.ClientID]
	user.Deck = deck
	server.Users[request.ClientID] = user

	resp := shared.Response{
            Status: "success",
            Action: "CHANGE_DECK",
            Server: server.ID,
        }
	data, _ := json.Marshal(resp)
	nc.Publish(msg.Reply, data)
	fmt.Printf("%s deck atualizado\n", request.ClientID)
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

//PARTE GLOBAL
var ActiveGames = make(map[string]*shared.GameRoom)

// Handler para iniciar partidas
func StartGameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var room shared.GameRoom
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		log.Printf("[StartGameHandler] Erro ao decodificar GameRoom: %v", err)
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	// Salva a partida ativa no servidor
	ActiveGames[room.ID] = &room
	log.Printf("[StartGameHandler] Nova partida recebida: %s (%s vs %s). Turno: %s",
		room.ID, room.Player1.UserName, room.Player2.UserName, room.Turn)

	// Aqui você pode inicializar timers, preparar decks, ou iniciar a lógica do jogo
	// Exemplo:
	// go StartMatch(room.ID)

	// Responde para confirmar que recebeu
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{
		"status":  "success",
		"message": "Partida iniciada no host",
		"roomID":  room.ID,
	}
	json.NewEncoder(w).Encode(resp)
}

func StartHeartbeatMonitor(server *models.Server, nc *nats.Conn) {
	go func() {
		for {
			time.Sleep(heartbeatInterval)
			now := time.Now()

			mu.Lock()
			for id, c := range activeClients {
				// Se passou muito tempo desde o último pong
				if now.Sub(c.LastSeen) > disconnectTimeout {
					log.Printf("Cliente '%s' inativo por %v. Desconectando...", id, now.Sub(c.LastSeen))
					DisconnectClient(server, id)
					delete(activeClients, id)
					continue
				}

				// Se estiver em estado ativo, envia ping
				if c.State == Active {
					c.State = WaitingReconnection
					nc.Publish(fmt.Sprintf("client.%s.ping", id), []byte("ping"))
				}
			}
			mu.Unlock()
		}
	}()
}

// Chamado quando o cliente responde ao ping (pong)
func HandlePong(serverID int, clientID string) {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := activeClients[clientID]; ok {
		c.LastSeen = time.Now()
		c.State = Active
	}
}

func HandlePing(server *models.Server, req shared.Request, nc *nats.Conn, msg *nats.Msg) {
	clientID := req.ClientID
	// Atualiza último ping
	mu.Lock()
	if c, ok := activeClients[clientID]; ok {
		c.LastSeen = time.Now()
		c.State = Active
	}
	mu.Unlock()

	// Responde com PONG
	response := shared.Response{
		Status: "success",
		Action: "PONG",
	}
	respData, _ := json.Marshal(response)
	clientTopic := fmt.Sprintf("client.%s.inbox", clientID)
	nc.Publish(clientTopic, respData)
}


func HandleHeartbeat(serverID int, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
	clientID := request.ClientID

	mu.Lock()
	if c, ok := activeClients[clientID]; ok {
		c.LastSeen = time.Now()
		c.State = Active
	} else {
		activeClients[clientID] = &ClientInfo{
			ClientID: clientID,
			LastSeen: time.Now(),
			State:    Active,
		}
	}
	mu.Unlock()
}

/*// Heatbeat para o clinete
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
					//nc.Publish("server."+strconv.Itoa(server.ID)+".requests", data)
					nc.Publish(fmt.Sprintf("client.%s.inbox", id), data)

				}
			}
			mu.Unlock()
		}
	}()
}
*/

/*// Handler para processar os heartbeats recebidos via NATS
func HandleHeartbeat(serverID int, request shared.Request, nc *nats.Conn, msg *nats.Msg) {
	clientID := request.ClientID
	mu.Lock()
	activeClients[clientID] = &ClientInfo{
		ClientID: clientID,
		LastSeen: time.Now(),
	}
	mu.Unlock()

	//log.Printf("[%d] - Heartbeat recebido de %s", serverID, clientID)

}*/


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
