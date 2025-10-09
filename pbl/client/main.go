package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"pbl/client/utils"
	"pbl/shared"

	"github.com/nats-io/nats.go"
)

type ServerInfo struct {
    ID   int
    Name string
}

func main() {
	//Conecta no NATS
	nc, err := nats.Connect("nats://localhost:4222") // Para rodar local
	
	//nc, err := nats.Connect("nats://nats:4222")   // Para Docker 

	if err != nil {
		log.Fatalf("Erro ao conectar no NATS: %v", err)
	}
	defer nc.Close()

	log.Println("Conectado ao NATS!")

	//TODO: substituir hardcoded servers por descoberta dinâmica via Pub/Sub (veremos)
	servers := []ServerInfo{
    	{ID: 1, Name: "Servidor 1"},
        {ID: 2, Name: "Servidor 2"},
        {ID: 3, Name: "Servidor 3"},
    }

	chooseString := utils.EscolherServidor()
	chooseInt, err := strconv.Atoi(chooseString)
	if err != nil{
		fmt.Println("Erro ao converter")
		return
	}

	chosenServer := servers[chooseInt-1]
	fmt.Printf("Você escolheu: %s (ID=%d)\n", chosenServer.Name, chosenServer.ID)
	//Publica no tópico de escolher o servidor
	
	// Monta payload da escolha do servidor
	payload := map[string]int{"server_id": chosenServer.ID}

	// Converte para JSON e transforma em json.RawMessage
	payloadJSON, _ := json.Marshal(payload)

	// Cria Request
	req := shared.Request{
		ClientID: "cliente1", 
		Action:   "CHOOSE_SERVER",
		Payload:  json.RawMessage(payloadJSON), 
	}

	//Converte Request em JSON
	dataReq, _ := json.Marshal(req)

	// Publica no tópico do servidor escolhido
	topic := fmt.Sprintf("server.%d.requests", chosenServer.ID)
	err = nc.Publish(topic, dataReq)
	if err != nil {
		log.Printf("Erro ao publicar escolha do servidor: %v", err)
	} else {
		log.Printf("Escolha do servidor enviada com sucesso!")
	}
	
	utils.MenuInicial()

	//Publica no tópico CADASTRO
	data := utils.Cadastro()
	jsonData, err := json.Marshal(data)
	if err != nil {
    	log.Printf("Erro ao converter para JSON: %v", err)
    	return
	}

	err = nc.Publish("REGISTER", jsonData)
	if err != nil {
		log.Printf("Erro ao publicar mensagem: %v", err)
	}

	log.Printf("Cadastro enviado com sucesso!")
	
}