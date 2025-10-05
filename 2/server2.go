package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"message": "Hello from Server 2"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	json.NewDecoder(r.Body).Decode(&data)
	response := map[string]interface{}{"received_by_server2": data}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendToServer1() {
	for {
		// Tenta GET no Server 1
		resp, err := http.Get("http://localhost:8080/hello")
		if err != nil {
			fmt.Println("Server 2: Server 1 ainda não disponível, tentando de novo...")
			time.Sleep(1 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Server 2 recebeu GET de Server 1:", string(body))
		resp.Body.Close()

		// Tenta POST no Server 1
		msg := map[string]string{"msg": "Hello from Server 2"}
		jsonData, _ := json.Marshal(msg)

		resp2, err := http.Post("http://localhost:8080/send", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Server 2: Erro no POST para Server 1, tentando de novo...")
			time.Sleep(1 * time.Second)
			continue
		}
		body2, _ := io.ReadAll(resp2.Body)
		fmt.Println("Server 2 recebeu POST de Server 1:", string(body2))
		resp2.Body.Close()

		// Espera antes de enviar de novo
		time.Sleep(5 * time.Second)
	}
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/send", sendHandler)

	// Sobe o servidor
	go func() {
		fmt.Println("Server 2 rodando na porta 8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			fmt.Println("Erro Server 2:", err)
		}
	}()

	// Começa a enviar mensagens para Server 1
	sendToServer1()
}
