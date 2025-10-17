package models

import (
	"pbl/shared"
	"sync"

	"github.com/hashicorp/raft"
)

type Server struct {
	ID      int
	Port    string
	SelfURL string
	Peers   []PeerInfo
	Raft    *raft.Raft
	Users   map[string]shared.User
	Mu      sync.Mutex
}

type Message struct {
	From    int    `json:"from"`
	MsgType string `json:"msg_type"`
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
