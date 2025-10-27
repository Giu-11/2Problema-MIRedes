package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pbl/server/fsm"
	"pbl/server/handlers"
	"pbl/server/models"
	"pbl/server/pubSub"
	"pbl/server/utils"
	"pbl/shared"
	"pbl/style"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

func StartServer(idString, port, peersEnv, natsURL string) error {
	style.Clear()
	id, _ := strconv.Atoi(idString)
	if port == "" {
		port = "8001"
	}

	peerInfos := parsePeers(peersEnv)
	server := models.NewServer(id, port, peerInfos)

	
	// Configuração Raft
	config := raft.DefaultConfig()
	config.HeartbeatTimeout = 2000 * time.Millisecond  // ← Aumenta timeouts
	config.ElectionTimeout = 2000 * time.Millisecond
	config.CommitTimeout = 1000 * time.Millisecond

	config.LocalID = raft.ServerID(idString)

	raftAdvAddr := os.Getenv("RAFT_ADVERTISE_ADDR")
		if raftAdvAddr == "" {
			// Fallback para localhost se não estiver no Docker (para rodar local)
			log.Printf("RAFT_ADVERTISE_ADDR não definida, usando fallback para localhost:%s", port)
			raftAdvAddr = "localhost:" + port
		}

	//raftListenAddr := "0.0.0.0:" + port
	transport := NewHTTPTransport(raft.ServerAddress(raftAdvAddr))

	
	dataDir := filepath.Join(".", "raft_data", idString)
	os.MkdirAll(dataDir, 0700)

	snapshots, err := raft.NewFileSnapshotStore(dataDir, 2, os.Stderr)
	if err != nil {
		return err
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
	if err != nil {
		return err
	}

	fsm := fsm.NewFSM()
	server.FSM = fsm // importante: associar FSM ao servidor

	ra, err := raft.NewRaft(config, fsm, logStore, logStore, snapshots, transport)
	if err != nil {
		return err
	}
	server.Raft = ra
	fsm.Raft = ra // FSM precisa do Raft para checar se é líder


	// Bootstrap cluster
	var configuration raft.Configuration
	
	// CORRIGIDO: Use o endereço de anúncio!
	selfAddr := raftAdvAddr 
	configuration.Servers = []raft.Server{
		{ID: raft.ServerID(idString), Address: raft.ServerAddress(selfAddr)},
	}
	// Bootstrap cluster
	for _, peer := range peerInfos {
		// peer.URL vem como "http://server2:8002"
		// Precisamos apenas de "server2:8002"
		raftPeerAddr := strings.TrimPrefix(peer.URL, "http://")
		raftPeerAddr = strings.TrimPrefix(raftPeerAddr, "https://") 
		
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      raft.ServerID(strconv.Itoa(peer.ID)),
			Address: raft.ServerAddress(raftPeerAddr), // <--- CORRIGIDO
		})
	}
	ra.BootstrapCluster(configuration)

	// Inicia NATS
	nc, err := pubSub.StartNats(server)
	if err != nil {
		log.Fatalf("Erro NATS: %v", err)
	}
	server.Matchmaking.Nc = nc

	// Monitora fila local e heartbeat
	go handlers.MonitorLocalQueue(server, nc)
	log.Printf("[Servidor %d] Monitor de matchmaking local iniciado.", server.ID)
	handlers.StartHeartbeatMonitor(server, nc)

	// Ticker para o líder tentar criar partidas a cada 500ms
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if server.Raft != nil && server.Raft.State() == raft.Leader {
				//log.Println("[LÍDER] Verificando fila global para criar partidas...")
				fsm.TryMatchPlayers()
			}
		}
	}()

	// Goroutine para escutar salas criadas e notificar outros servidores
	go func() {
		for room := range server.FSM.CreatedRooms {
			// Apenas o líder envia notificações
			if server.FSM.Raft != nil && server.FSM.Raft.State() == raft.Leader {
				go notifyServersAboutMatch(room, server)
				go utils.NotifyClients(*room, server)
			}
		}
	}()

	// HTTP handlers
	http.HandleFunc("/raft", transport.HandleRaftRequest)
	http.HandleFunc("/leader/draw-card", handlers.LeaderDrawCardHandler(server))
	http.HandleFunc("/leader/join-global-queue", handlers.LeaderJoinGlobalQueueHandler(server))
	
	http.HandleFunc("/notify-match", func(w http.ResponseWriter, r *http.Request) {
		var room shared.GameRoom
		if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		utils.NotifyClients(room, server)
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/forward-card", handlers.HandleForwardCard(server, nc))
	http.HandleFunc("/forward-result", handlers.HandleForwardCard(server, nc))
	http.HandleFunc("/forward-to-host", handlers.HandleForwardToHost(server, nc))


	log.Printf("[Servidor %d] HTTP iniciado na porta %s, pronto para Raft e NATS", server.ID, server.Port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Erro no servidor HTTP: %v", err)
	}

	return nil
}


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

func notifyServersAboutMatch(room *shared.GameRoom, server *models.Server) {
    payload, err := json.Marshal(room)
    if err != nil {
        log.Printf("[Notify] Erro ao serializar sala: %v", err)
        return
    }

    for _, peer := range server.Peers {
        url := peer.URL + "/notify-match"
        go func(url string) {
            resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
            if err != nil {
                log.Printf("[Notify] Erro ao enviar para %s: %v", url, err)
                return
            }
            resp.Body.Close()
            log.Printf("[Notify] Notificação enviada para %s", url)
        }(url)
    }
}


