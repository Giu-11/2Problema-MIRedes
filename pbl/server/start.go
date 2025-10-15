package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pbl/server/models"
	"pbl/server/pubSub"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

func StartServer(idString, port, peersEnv, natsURL string) error {
	id, _ := strconv.Atoi(idString)
	if port == "" {
		port = "8001"
	}
	peerInfos := parsePeers(peersEnv)

	server := &models.Server{
		ID:    id,
		Port:  port,
		Peers: peerInfos,
	}

	// --- CONFIGURAÇÃO DO RAFT com Transporte HTTP ---
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(idString)

	// O endereço para o transporte Raft agora é o mesmo do nosso servidor HTTP
	raftAddr := "localhost:" + port
	
	// Cria nosso transporte customizado
	transport := NewHTTPTransport(raft.ServerAddress(raftAddr))
	
	// O resto da configuração do Raft é igual (snapshots, log store, FSM)
	dataDir := filepath.Join(".", "raft_data", idString)
	os.MkdirAll(dataDir, 0700)

	snapshots, err := raft.NewFileSnapshotStore(dataDir, 2, os.Stderr)
	if err != nil { return err }

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
	if err != nil { return err }

	fsm := NewFSM()

	ra, err := raft.NewRaft(config, fsm, logStore, logStore, snapshots, transport)
	if err != nil { return err }
	server.Raft = ra

	// Bootstrap do cluster
	var configuration raft.Configuration
	// Adiciona o próprio servidor
	configuration.Servers = []raft.Server{
		{
			ID:      raft.ServerID(idString),
			Address: transport.LocalAddr(),
		},
	}
	// Adiciona os outros peers
	for _, peer := range peerInfos {
		peerAddr := "localhost:" + strconv.Itoa(8000+peer.ID) // Ex: localhost:8002
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      raft.ServerID(strconv.Itoa(peer.ID)),
			Address: raft.ServerAddress(peerAddr),
		})
	}
	ra.BootstrapCluster(configuration)
	
	// --- SERVIÇOS ANTIGOS ---
	// Conexão NATS continua a mesma
	_, err = pubSub.StartNats(server)
	if err != nil { log.Fatalf("Erro NATS: %v", err) }

	// --- REGISTRO DOS HANDLERS HTTP ---
	// Endpoint REST para o protocolo Raft
	http.HandleFunc("/raft", transport.HandleRaftRequest)
	// Endpoints antigos que você quer manter
	// O handler de register agora precisa do objeto `server` para acessar o Raft
	// (você precisará ajustar a assinatura em pubSub.go e handlers.go)
	// http.HandleFunc("/register", handlers.HandleRegister(server))
	
	log.Printf("[%d] - Servidor HTTP iniciado na porta %s, pronto para Raft e NATS", server.ID, server.Port)
	if err := http.ListenAndServe(":"+server.Port, nil); err != nil {
		log.Fatalf("Erro no servidor HTTP: %v", err)
	}

	return nil
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
