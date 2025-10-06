package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"pbl/server/handlers"
	"pbl/server/models"
	"pbl/server/utils"
)

func main() {
	//Detecta IP
	ip, err := utils.LocalIP()
	if err != nil {
		log.Printf("Não foi possível detectar IP: %v", err)
		ip = "127.0.0.1"
	}

	log.Println("IP detectado:", ip)

	//Gera ID aleatório
	id := utils.GerarIdAleatorio()
	log.Printf("ID gerado: %d", id)

	/*//Listener em porta automática
	serverHTTP := &http.Server{Handler: nil}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	log.Printf("Porta escolhida automaticamente: %s", port)
	fmt.Println(serverHTTP)*/
	
	port := os.Getenv("PORT")
	if port == "" {
    	port = "8001"
	}


	// URL do próprio servidor
	selfURL := fmt.Sprintf("http://%s:%s", ip, port)
	log.Printf("Meu URL: %s", selfURL)

	// Lê peers da env
	peersEnv := os.Getenv("PEERS")
	peers := []string{}
	if peersEnv != "" {
		peers = strings.Split(peersEnv, ",")
	}

	// Converte em []PeerInfo
	peerInfos := []models.PeerInfo{}
	for i, url := range peers {
		peerInfos = append(peerInfos, models.PeerInfo{
			ID:  i + 1,
			URL: url,
		})
	}

	// Cria server
	server := models.Server{
		ID:    id,
		Port:  port,
		Peers: peerInfos,
	}

	// Registra handler
	http.HandleFunc("/ping", handlers.PingHandler(server.ID))

	log.Printf("Peers conhecidos: %+v", server.Peers)
	time.Sleep(3 * time.Second)
	go handlers.StartElection(&server)

	//Rotina de heartbeat
	go func() {
		for {
			for _, peer := range server.Peers {
				log.Println("Líder: ", server.Leader)
				msg := models.Message{
					From: server.ID,
					Msg:  "PING",
				}
				data, _ := json.Marshal(msg)

				resp, err := http.Post(peer.URL+"/ping", "application/json", strings.NewReader(string(data)))
				if err != nil {
					log.Printf("[%d] Erro ao pingar %s: %v", server.ID, peer.URL, err)
					continue
				}
				resp.Body.Close()
			}
			time.Sleep(5 * time.Second)
		}
	}()

	log.Printf("[%d] Servidor iniciado na porta %s", server.ID, server.Port)
	log.Fatal(http.ListenAndServe(":"+server.Port, nil))

}
