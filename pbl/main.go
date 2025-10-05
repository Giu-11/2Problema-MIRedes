package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var port string
var peers []string

//Estrutura do payload
type Message struct {
	From string `json:"from"`
	Msg  string `json:"msg"`
}

//Handler para responder Pong
func pingHandler(w http.ResponseWriter, r *http.Request) {
	//Lê o corpo JSON (se houver)
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var msg Message
	if len(body) > 0 {
		json.Unmarshal(body, &msg)
	}

	//Log do que recebeu
	log.Printf("Recebi PING de %s", msg.From)

	//Resposta Pong
	resp := Message{
		From: "server:" + port,
		Msg:  "PONG",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	//Ler variáveis de ambiente
	port = os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}
	peersEnv := os.Getenv("PEERS")
	if peersEnv != "" {
		peers = strings.Split(peersEnv, ",")
	}

	//Rota /ping
	http.HandleFunc("/ping", pingHandler)

	//Rotina que envia PING periodicamente
	go func() {
		for {
			for _, peer := range peers {
				msg := Message{
					From: "server:" + port,
					Msg:  "PING",
				}
				data, _ := json.Marshal(msg)

				resp, err := http.Post(peer+"/ping", "application/json", strings.NewReader(string(data)))
				if err != nil {
					log.Printf("Erro ao pingar %s: %v", peer, err)
					continue
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				var reply Message
				json.Unmarshal(body, &reply)
				log.Printf("Resposta de %s → %+v", peer, reply)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	log.Printf("Servidor iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
