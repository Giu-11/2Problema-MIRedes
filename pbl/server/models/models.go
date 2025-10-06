package models

import (
    "sync"
)

type Server struct {
	ID       int64
	Port     string
	Peers    []PeerInfo
	IsLeader bool
	Leader   int
	mu       sync.RWMutex
	electionInProgress bool
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