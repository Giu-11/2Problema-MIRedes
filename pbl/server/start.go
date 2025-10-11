package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pbl/server/handlers"
	"pbl/server/models"
	"pbl/server/pubSub"
	"pbl/server/utils"
)

func StartServer(idString, port, peersEnv, natsURL string) error {
	ip, err := utils.LocalIP()
	if err != nil {
		log.Printf("Não foi possível detectar IP: %v", err)
		ip = "127.0.0.1"
	}
	//log.Println("IP detectado:", ip)

	// Define ID
	var id int
	if idString != "" {
		id, _ = strconv.Atoi(idString)
	}
	//log.Printf("ID do servidor: %d", id)

	if port == "" {
		port = "8001"
	}

	//Constrói lista de peers
	peerInfos := parsePeers(peersEnv)

	//Cria struct Server
	server := models.NewServer(id, port, peerInfos, ip)
	//log.Printf("Meu URL: %s", server.SelfURL)
	//log.Printf("Peers conhecidos: %+v", server.Peers)

	//Conecta ao NATS e inicia inscrição nos tópicos
	_, err = pubSub.StartNats(server)
	if err != nil {
		log.Fatalf("Erro ao inscrever no tópico NATS: %v", err)
	}
	log.Println("Conectado ao NATS:", natsURL)

	//Inscreve o servidor no tópico NATS
	_, err = pubSub.StartNats(server)
	if err != nil {
		log.Fatalf("Erro ao inscrever no tópico NATS: %v", err)
	}

	//Registra Handlers HTTP
	http.HandleFunc("/ping", handlers.PingHandler(server.ID))
	http.HandleFunc("/election", handlers.ElectionHandler(server))

	//Inicia eleição inicial (após um tempo)
	time.Sleep(5 * time.Second)
	go handlers.StartElection(server)

	//Inicia rotina de heartbeat
	go startHeartbeat(server)

	//Inicia servidor HTTP
	go func() {
		//log.Printf("[%d] - Servidor HTTP iniciado na porta %s", server.ID, server.Port)
		if err := http.ListenAndServe(":"+server.Port, nil); err != nil {
			log.Fatalf("Erro no servidor HTTP: %v", err)
		}
	}()

	select {}
}

// Função auxiliar para converter variável PEERS
func parsePeers(peersEnv string) []models.PeerInfo {
	var peers []models.PeerInfo
	if peersEnv == "" {
		return peers
	}
	pairs := strings.Split(peersEnv, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			peerID, _ := strconv.Atoi(parts[0])
			peers = append(peers, models.PeerInfo{ID: peerID, URL: parts[1]})
		}
	}
	return peers
}

// Rotina de heartbeat (ping nos peers)
func startHeartbeat(server *models.Server) {
	for {
		time.Sleep(5 * time.Second)
		server.Mu.Lock()
		leader := server.Leader
		server.Mu.Unlock()

		log.Printf("\n[%d] - Líder atual: %d", server.ID, leader)

		for _, peer := range server.Peers {
			msg := models.Message{
				From: server.ID,
				Msg:  "PING",
			}
			data, _ := json.Marshal(msg)
			resp, err := http.Post(peer.URL+"/ping", "application/json", strings.NewReader(string(data)))
			if err != nil {
				//log.Printf("[%d] - Erro ao pingar %s: %v", server.ID, peer.URL, err)
				continue
			}
			resp.Body.Close()
		}
	}
}
