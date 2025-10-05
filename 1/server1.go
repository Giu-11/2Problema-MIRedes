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
	response := map[string]string{"message": "Hello from Server 1"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	json.NewDecoder(r.Body).Decode(&data)
	response := map[string]interface{}{"received_by_server1": data}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Endpoints
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/send", sendHandler)

	// Sobe o servidor em goroutine
	go func() {
		fmt.Println("Server 1 rodando na porta 8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Erro Server 1:", err)
		}
	}()

	// Espera o Server 2 subir
	time.Sleep(2 * time.Second)

	// Faz GET no Server 2
	resp, err := http.Get("http://localhost:8081/hello")
	if err != nil {
		fmt.Println("Erro GET Server 2:", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Resposta GET de Server 2:", string(body))
		resp.Body.Close()
	}

	// Faz POST no Server 2
	msg := map[string]string{"msg": "Hello from Server 1"}
	jsonData, _ := json.Marshal(msg)

	resp2, err := http.Post("http://localhost:8081/send", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Erro POST Server 2:", err)
	} else {
		body2, _ := io.ReadAll(resp2.Body)
		fmt.Println("Resposta POST de Server 2:", string(body2))
		resp2.Body.Close()
	}

	// Mantém o programa rodando
	select {}
}
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
	response := map[string]string{"message": "Hello from Server 1"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]string
	json.NewDecoder(r.Body).Decode(&data)
	response := map[string]interface{}{"received_by_server1": data}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendToServer2() {
	for {
		// Tenta GET no Server 2
		resp, err := http.Get("http://localhost:8081/hello")
		if err != nil {
			fmt.Println("Server 1: Server 2 ainda não disponível, tentando de novo...")
			time.Sleep(1 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Server 1 recebeu GET de Server 2:", string(body))
		resp.Body.Close()

		// Tenta POST no Server 2
		msg := map[string]string{"msg": "Hello from Server 1"}
		jsonData, _ := json.Marshal(msg)

		resp2, err := http.Post("http://localhost:8081/send", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Server 1: Erro no POST para Server 2, tentando de novo...")
			time.Sleep(1 * time.Second)
			continue
		}
		body2, _ := io.ReadAll(resp2.Body)
		fmt.Println("Server 1 recebeu POST de Server 2:", string(body2))
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
		fmt.Println("Server 1 rodando na porta 8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Erro Server 1:", err)
		}
	}()

	// Começa a enviar mensagens para Server 2
	sendToServer2()
}
