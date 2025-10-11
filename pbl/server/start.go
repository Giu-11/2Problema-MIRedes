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
		// Intervalo um pouco maior para dar tempo de resposta
		time.Sleep(7 * time.Second) 

		// Bloqueia o Mutex no início para ler e usar os dados do servidor de forma segura
		server.Mu.Lock()
		currentLeaderID := server.Leader
		electionInProgress := server.ElectionInProgress
		peers := server.Peers
		server.Mu.Unlock() // Desbloqueia logo após a leitura

		// Se este servidor for o líder, ele não precisa pingar ninguém para detectar falha do líder
		// ou se uma eleição já estiver em andamento, espera a próxima rodada.
		if server.IsLeader || electionInProgress {
			continue
		}

		// fica com o heartbeat so para o lider
		log.Printf("\n[%d] - Verificando líder. Líder atual: %d", server.ID, currentLeaderID)

		// Encontra as informações do peer que é o líder atual
		var leaderPeer models.PeerInfo
		foundLeader := false
		for _, p := range peers {
			if p.ID == currentLeaderID {
				leaderPeer = p
				foundLeader = true
				break
			}
		}

		// Se o líder não for encontrado na lista de peers ou for ele mesmo, não faz nada.
		if !foundLeader || currentLeaderID == server.ID {
			continue
		}

		// Tenta pingar o líder
		msg := models.Message{
			From: server.ID,
			Msg:  "PING",
		}
		data, _ := json.Marshal(msg)
		_, err := http.Post(leaderPeer.URL+"/ping", "application/json", strings.NewReader(string(data)))

		// Se der erro ao pingar o líder...
		if err != nil {
			log.Printf("[%d] - ERRO ao pingar o líder %d (%s): %v. INICIANDO NOVA ELEIÇÃO!", server.ID, leaderPeer.ID, leaderPeer.URL, err)
			
			// Inicia uma nova eleição em uma goroutine para não bloquear o heartbeat
			go handlers.StartElection(server) 
		}
	}
}
