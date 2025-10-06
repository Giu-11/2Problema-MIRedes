package models

import (
    "sync"
)
type Server struct {
    ID                int
    Port              string
    SelfURL           string
    Peers             []PeerInfo
    IsLeader          bool
    Leader            int
    Mu                sync.Mutex
    ElectionInProgress bool
    ReceivedOK bool
}

type Message struct {
    From int `json:"from"`
    MsgType string `json:"msg_type"`
    Msg     string `json:"msg"`
}

type PeerInfo struct {
	ID  int
	URL string
}

type ElectionMessage struct{
	Type string `json:"type"`
	FromID int `json:"from_id"`     
	FromURL string `json:"leader_url,omitempty"`
	LeaderID int    `json:"leader_id"`
}