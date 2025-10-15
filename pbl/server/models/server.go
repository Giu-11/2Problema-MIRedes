package models

import (
	"fmt"
)

func NewServer(id int, port string, peers []PeerInfo, ip string) *Server {
    return &Server{
        ID:                 id,
        Port:               port,
        Peers:              peers,
        SelfURL:            fmt.Sprintf("http://%s:%s", ip, port),
    }
}
