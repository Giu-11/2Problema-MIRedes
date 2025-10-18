package handlers

import (
	"encoding/json"
	"log"
	"pbl/server/game"
	"pbl/server/models"
	sharedRaft "pbl/server/shared"
	"pbl/shared"
	"time"

	"github.com/hashicorp/raft"
	"github.com/nats-io/nats.go"
)

//Colocar cliente na fila --> fila local
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

//Monitora a fila local e move jogadores para a fila global se passarem de 5s
func MonitorLocalQueue(server *models.Server, nc *nats.Conn) {
	ticker := time.NewTicker(1 * time.Second)

	for range ticker.C {
		now := time.Now()

		server.Matchmaking.Mutex.Lock()

		//Move jogadores da fila local para global se esperaram demais
		for i := 0; i < len(server.Matchmaking.LocalQueue); {
			entry := server.Matchmaking.LocalQueue[i]
			waitTime := now.Sub(entry.JoinTime)

			if waitTime > 5*time.Second {
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
		MatchLocalQueue(server)

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
		log.Printf("[Líder] Cliente %s adicionado à fila global diretamente", entry.Player.UserName)
	}
}

//Adiciona cliente na fila global do líder e replica via Raft
func handleJoinGlobalQueueLeader(entry shared.QueueEntry, server *models.Server) {
	// Protege acesso à fila global
	server.Matchmaking.Mutex.Lock()
	server.Matchmaking.GlobalQueue = append(server.Matchmaking.GlobalQueue, entry)
	server.Matchmaking.Mutex.Unlock()

	log.Printf("[DEBUG] Fila local atual: %v", ListLocalQueue(server))
	//Aplica comando Raft para replicação
	dataEntry, _ := json.Marshal(entry)
	cmd := sharedRaft.Command{
		Type: "JOIN_GLOBAL_QUEUE",
		Data: dataEntry,
	}
	data, _ := json.Marshal(cmd)
	server.Raft.Apply(data, 5*time.Second) //replica o estado
}

func MatchLocalQueue(server *models.Server){
	server.Matchmaking.Mutex.Lock()
	defer server.Matchmaking.Mutex.Unlock()

	for len(server.Matchmaking.LocalQueue) >= 2{
		player1 := server.Matchmaking.LocalQueue[0].Player
		player2 := server.Matchmaking.LocalQueue[1].Player

		if player1.Status != "free" || player2.Status != "free"{
			break
		}

		player1.Status = "jogando"
		player2.Status = "jogando"

		room := game.CreateRoom(player1, player2)
		log.Printf("\n[Server %d] - Nova partida craida: %s (%s vs %s)", server.ID, room.ID, player1.UserName, player2.UserName)

		server.Matchmaking.LocalQueue = server.Matchmaking.LocalQueue[2:]
	}
}

//listar as filas
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
