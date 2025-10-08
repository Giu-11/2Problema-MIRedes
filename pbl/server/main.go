package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"strconv"

	"github.com/nats-io/nats.go"

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

	/*//Gera ID aleatório
	id := utils.GerarIdAleatorio()
	log.Printf("ID gerado: %d", id)*/

	/*//Listener em porta automática
	serverHTTP := &http.Server{Handler: nil}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	log.Printf("Porta escolhida automaticamente: %s", port)
	fmt.Println(serverHTTP)*/
	
	idString := os.Getenv("ID")
	var id int
	if idString != "" {
		id, _ = strconv.Atoi(idString)
	} else {
		id = utils.GerarIdAleatorio() // Fallback
	}
	log.Printf("ID do servidor: %d", id)


	port := os.Getenv("PORT")
	if port == "" {
    	port = "8001"
	}

	//URL do próprio servidor
	selfURL := fmt.Sprintf("http://%s:%s", ip, port)
	log.Printf("Meu URL: %s", selfURL)

	//Lê peers da env
	peersEnv := os.Getenv("PEERS")
	//log.Printf("PEERS env: '%s'", peersEnv)
	peerInfos := []models.PeerInfo{}
	
	if peersEnv != "" {
		peerPairs := strings.Split(peersEnv, ",")
		for _, pair := range peerPairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				peerID, _ := strconv.Atoi(parts[0])
				peerInfos = append(peerInfos, models.PeerInfo{
					ID:  peerID,
					URL: parts[1],
				})
			}
		}
	}

	// Cria server
	server := &models.Server{
		ID:                 id,
		Port:               port,
		Peers:              peerInfos,
		SelfURL:            selfURL,
		ElectionInProgress: false,
		ReceivedOK:         false,
		Leader:             0,
		IsLeader:           false,
	}

	//Registra handlers
	http.HandleFunc("/ping", handlers.PingHandler(server.ID))
	http.HandleFunc("/election", handlers.ElectionHandler(server))

	log.Printf("Peers conhecidos: %+v", server.Peers)

	time.Sleep(5 * time.Second)
	go handlers.StartElection(server)

	//Rotina de heartbeat
	go func() {
		for {
			time.Sleep(5 * time.Second)
			
			server.Mu.Lock()
			leader := server.Leader
			server.Mu.Unlock()
			
			log.Printf("\n[%d] - Líder atual: %d", server.ID, leader)
			
			for _, peer := range server.Peers {
				msg := models.Message{
					From: server.ID,
					Msg:  "PING",
				}
				data, _ := json.Marshal(msg)

				resp, err := http.Post(peer.URL+"/ping", "application/json", strings.NewReader(string(data)))
				if err != nil {
					log.Printf("[%d] - Erro ao pingar %s: %v", server.ID, peer.URL, err)
					continue
				}
				resp.Body.Close()
			}
		}
	}()

	//Conexão com NATS
		
	//nc, err := nats.Connect("nats://localhost:4222") //pra localhost

	nc, err := nats.Connect("nats://nats:4222") //pra docker compose
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS: %v", err)
	}
	defer nc.Close()

	//servidor se inscreve no topico CADASTRO
	_, err = nc.Subscribe("CADASTRO", func(msg *nats.Msg) {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("[%d] Erro ao decodificar mensagem de cadastro: %v", server.ID, err)
			return
		}

		log.Printf("[%d] - Recebi pedido de cadastro: %+v", server.ID, payload)

	})
	if err != nil {
		log.Fatalf("Erro ao se inscrever no NATS: %v", err)
	}
	
	go func() {
		log.Printf("[%d] - Servidor HTTP iniciado na porta %s", server.ID, server.Port)
		if err := http.ListenAndServe(":"+server.Port, nil); err != nil {
			log.Fatalf("Erro no servidor HTTP: %v", err)
    	}
	}()

	select {}

}
