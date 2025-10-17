package models

import (
    "pbl/shared"
)

func NewServer(id int, port string, peers []PeerInfo) *Server {
    return &Server{
        ID:                 id,
        Port:               port,
        Peers:              peers,
        //SelfURL:            fmt.Sprintf("http://%s:%s", ip, port),
        Users: make(map[string]shared.User),
    }
}

