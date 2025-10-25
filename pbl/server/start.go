package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	
	"pbl/server/handlers"
	"pbl/server/models"
	"pbl/server/fsm"
	
	"pbl/server/pubSub"
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
	config.LocalID = raft.ServerID(idString)

	raftAddr := "0.0.0.0:" + port
	transport := NewHTTPTransport(raft.ServerAddress(raftAddr))

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
	selfAddr := "server" + idString + ":" + port
	configuration.Servers = []raft.Server{
		{ID: raft.ServerID(idString), Address: raft.ServerAddress(selfAddr)},
	}
	for _, peer := range peerInfos {
		peerAddr := "server" + strconv.Itoa(peer.ID) + ":" + strconv.Itoa(8000+peer.ID)
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      raft.ServerID(strconv.Itoa(peer.ID)),
			Address: raft.ServerAddress(peerAddr),
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

	// HTTP handlers
	http.HandleFunc("/raft", transport.HandleRaftRequest)
	http.HandleFunc("/leader/draw-card", handlers.LeaderDrawCardHandler(server))
	http.HandleFunc("/leader/join-global-queue", handlers.LeaderJoinGlobalQueueHandler(server))

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
