package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"pbl/server/models"
	"pbl/server/utils"
)

func main() {

	ip, err := utils.LocalIP()
	if err != nil {
		log.Printf("Não foi possível detectar o IP")
		ip = "127.0.0.1"
	}
	log.Println("IP detectado: ", ip)

	id := utils.GerarIdAleatorio()
	log.Printf("ID gerado: ", id)

	//Cria listener HTTP em porta automática
	serverHTTP := &http.Server{Handler: nil}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	log.Printf("Porta escolhida automaticamente: %s", port)

	//Monta URL do servidor
	selfURL := fmt.Sprintf("http://%s:%s", ip, port)
	log.Printf("Meu URL: %s", selfURL)

	peerInfos := []models.PeerInfo{}
	for i, url := range peersURLs {
		peerInfos = append(peerInfos, models.PeerInfo{
			ID:  int64(i + 1),
			URL: url,
		})
	}

	server := models.Server{
		ID:    id,
		Port:  port,
		Peers: peerInfos,
	}
}
