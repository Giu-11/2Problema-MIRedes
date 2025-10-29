package models

import (
    "pbl/shared"
)

func NewServer(id int, port string, peers []PeerInfo) *Server {
    return &Server{
        ID:                 id,
        Port:               port,
        Peers:              peers,
        Users: make(map[string]shared.User),
    }
}

