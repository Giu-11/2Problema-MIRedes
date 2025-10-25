package models

import (
	"sync"
	"pbl/shared"
	"github.com/hashicorp/raft"
	"github.com/nats-io/nats.go"
)

type Server struct {
	ID      int
	Port    string
	SelfURL string
	Peers   []PeerInfo
	Raft    *raft.Raft
	Users   map[string]shared.User
	Mu      sync.Mutex
	Matchmaking Matchmaking

}

type Message struct {
	From    int    `json:"from"`
	Type string `json:"msg_type"`
	Msg     string `json:"msg"`
}

type PeerInfo struct {
	ID  int
	URL string
}

type ElectionMessage struct {
	Type     string `json:"type"`
	FromID   int    `json:"from_id"`
	FromURL  string `json:"leader_url,omitempty"`
	LeaderID int    `json:"leader_id"`
}
type Matchmaking struct {
	LocalQueue  []shared.QueueEntry
	GlobalQueue []shared.QueueEntry //só o líder vai usar
	Mutex       sync.Mutex
	Nc          *nats.Conn   // conexão com NATS
	IsLeader    bool         // indica se este servidor é o líder
}
