package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"pbl/server/game"
	"pbl/server/models"
	sharedRaft "pbl/server/shared"
	"pbl/shared"
	"time"

	"github.com/hashicorp/raft"
	"github.com/nats-io/nats.go"
)

// Colocar cliente na fila --> fila local
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

	entry.JoinTime = time.Now() //sobreescreve o horário que o do cliente pode estar com erro

	//Adiciona na fila local
	server.Matchmaking.Mutex.Lock()
	server.Matchmaking.LocalQueue = append(server.Matchmaking.LocalQueue, entry)
	server.Matchmaking.Mutex.Unlock()

	log.Printf("Cliente %s entrou na fila local do servidor %d", entry.Player.UserName, server.ID)

	log.Printf("[DEBUG] Fila local atual: %v", ListLocalQueue(server))

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

// Monitora a fila local e move jogadores para a fila global se passarem de 5s
func MonitorLocalQueue(server *models.Server, nc *nats.Conn) {
	ticker := time.NewTicker(1 * time.Second)

	for range ticker.C {
		//log.Printf("[Matchmaking] Verificando fila local...")
		now := time.Now()

		server.Matchmaking.Mutex.Lock()

		//Move jogadores da fila local para global se esperaram demais
		for i := 0; i < len(server.Matchmaking.LocalQueue); {
			entry := server.Matchmaking.LocalQueue[i]
			waitTime := now.Sub(entry.JoinTime)

			if waitTime > 10*time.Second {
				sendToGlobalQueue(entry, server, nc)
				server.Matchmaking.LocalQueue = append(
					server.Matchmaking.LocalQueue[:i],
					server.Matchmaking.LocalQueue[i+1:]...,
				)
			} else {
				i++
			}
		}

		server.Matchmaking.Mutex.Unlock()

		//Faz matchmaking local
		MatchLocalQueue(server, nc)

		//tem que implementat a parte global

	}
}

// Envia o jogador ao líder
func sendToGlobalQueue(entry shared.QueueEntry, server *models.Server, nc *nats.Conn) {
	isLeader := server.Raft.State() == raft.Leader
	leaderAddr := string(server.Raft.Leader())

	if !isLeader {
		if leaderAddr == "" {
			log.Println("Nenhum líder disponível no momento")
			return
		}

		payload, _ := json.Marshal(entry)
		req := shared.Request{
			Action:  "JOIN_GLOBAL_QUEUE",
			Payload: payload,
		}
		data, _ := json.Marshal(req)

		// envia pro líder via NATS
		err := nc.Publish("leader.queue.join", data)
		if err != nil {
			log.Printf("Erro ao enviar cliente %s ao líder: %v", entry.Player.UserName, err)
		}
	} else {
		handleJoinGlobalQueueLeader(entry, server)
		log.Printf("[Líder] - Cliente %s adicionado à fila global diretamente", entry.Player.UserName)
	}
}

// Adiciona cliente na fila global do líder e replica via Raft
func handleJoinGlobalQueueLeader(entry shared.QueueEntry, server *models.Server) {
	// Protege acesso à fila global
	server.Matchmaking.Mutex.Lock()
	server.Matchmaking.GlobalQueue = append(server.Matchmaking.GlobalQueue, entry)
	server.Matchmaking.Mutex.Unlock()

	log.Printf("[DEBUG] Fila global atual: %v", ListGlobalQueue(server))
	//Aplica comando Raft para replicação
	dataEntry, _ := json.Marshal(entry)
	cmd := sharedRaft.Command{
		Type: "JOIN_GLOBAL_QUEUE",
		Data: dataEntry,
	}
	data, _ := json.Marshal(cmd)
	server.Raft.Apply(data, 5*time.Second) //replica o estado
}

func MatchLocalQueue(server *models.Server, nc *nats.Conn) {
	server.Matchmaking.Mutex.Lock()
	defer server.Matchmaking.Mutex.Unlock()

	for len(server.Matchmaking.LocalQueue) >= 2 {
		player1 := server.Matchmaking.LocalQueue[0].Player
		player2 := server.Matchmaking.LocalQueue[1].Player

		if player1.Status != "available" || player2.Status != "available" {
			break
		}

		player1.Status = "playing"
		player2.Status = "playing"

		//cria sala
		room := game.CreateRoom(&player1, &player2, nc, server.ID)
		log.Printf("\n[Server %d] - Nova partida criada: %s (%s vs %s)",
			server.ID, room.ID, player1.UserName, player2.UserName)

		//remove da fila
		server.Matchmaking.LocalQueue = server.Matchmaking.LocalQueue[2:]

		//envia mensagem MATCH para os dois clientes
		sendMatchNotification(server, room)
	}
}

func sendMatchNotification(server *models.Server, room *shared.GameRoom) {
	data, err := json.Marshal(room)
	if err != nil {
		log.Printf("Erro ao serializar room: %v", err)
		return
	}

	resp := shared.Response{
		Action: "MATCH",
		Status: "success",
		Data:   data,
		Server: server.ID,
	}

	msgData, _ := json.Marshal(resp)
	nc := server.Matchmaking.Nc

	// Notifica ambos os jogadores sobre o match
	topic1 := fmt.Sprintf("client.%s.inbox", room.Player1.UserId)
	topic2 := fmt.Sprintf("client.%s.inbox", room.Player2.UserId)

	nc.Publish(topic1, msgData)
	nc.Publish(topic2, msgData)

	log.Printf("[Server %d] - Match enviado para %s e %s",
		server.ID, room.Player1.UserName, room.Player2.UserName)
	log.Printf("[Server %d] - Turno inicial: %s", server.ID, room.Turn)
}

// listar as filas
func ListLocalQueue(server *models.Server) []string {
	server.Matchmaking.Mutex.Lock()
	defer server.Matchmaking.Mutex.Unlock()

	users := make([]string, len(server.Matchmaking.LocalQueue))
	for i, entry := range server.Matchmaking.LocalQueue {
		users[i] = entry.Player.UserName
	}
	return users
}

func ListGlobalQueue(server *models.Server) []string {
	server.Matchmaking.Mutex.Lock()
	defer server.Matchmaking.Mutex.Unlock()

	users := make([]string, len(server.Matchmaking.GlobalQueue))
	for i, entry := range server.Matchmaking.GlobalQueue {
		users[i] = entry.Player.UserName
	}
	return users
}
