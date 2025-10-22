package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pbl/server/handlers"
	"pbl/server/models"
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

	//configuração do raft com Transporte HTTP
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(idString)

	//O endereço para o transporte Raft
	raftAddr := "0.0.0.0:" + port
	transport := NewHTTPTransport(raft.ServerAddress(raftAddr))

	//O resto da configuração do Raft (snapshots, log store, FSM)
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

	fsm := NewFSM()

	ra, err := raft.NewRaft(config, fsm, logStore, logStore, snapshots, transport)
	if err != nil {
		return err
	}
	server.Raft = ra

	//bootstrap do cluster
	var configuration raft.Configuration
	//Adiciona o próprio servidor
	selfAddr := "server" + idString + ":" + port
	configuration.Servers = []raft.Server{
		{
			ID:      raft.ServerID(idString),
			Address: raft.ServerAddress(selfAddr),
		},
	}
	// Adiciona os outros peers
	for _, peer := range peerInfos {
		// Usa o nome do serviço do Docker Compose
		peerAddr := "server" + strconv.Itoa(peer.ID) + ":" + strconv.Itoa(8000+peer.ID)
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      raft.ServerID(strconv.Itoa(peer.ID)),
			Address: raft.ServerAddress(peerAddr),
		})
	}
	ra.BootstrapCluster(configuration)

	nc, err := pubSub.StartNats(server)
	if err != nil {
		log.Fatalf("Erro NATS: %v", err)
	}

	//Para rodar em background até acontecer o match
	server.Matchmaking.Nc = nc
	go handlers.MonitorLocalQueue(server, nc)
	log.Printf("[Servidor %d] Monitor de matchmaking iniciado.", server.ID)


	handlers.StartHeartbeatMonitor(server, nc)

	http.HandleFunc("/raft", transport.HandleRaftRequest)

	http.HandleFunc("/leader/draw-card", handlers.LeaderDrawCardHandler(server))

	log.Printf("[%d] - Servidor HTTP iniciado na porta %s, pronto para Raft e NATS", server.ID, server.Port)
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
