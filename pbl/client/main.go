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
	NATS string
}

func main() {
	//TODO: substituir hardcoded servers por descoberta dinâmica via Pub/Sub (veremos)
	//agora tá conectando por localhost mas tem que mudar isso depois
	servers := []ServerInfo{
		{ID: 1, Name: "Servidor 1", NATS: "nats://localhost:4223"},
		{ID: 2, Name: "Servidor 2", NATS: "nats://localhost:4224"},
		{ID: 3, Name: "Servidor 3", NATS: "nats://localhost:4225"},
	}

	chooseString := utils.EscolherServidor()
	chooseInt, err := strconv.Atoi(chooseString)
	if err != nil {
		fmt.Println("Erro ao converter")
		return
	}

	chosenServer := servers[chooseInt-1]
	fmt.Printf("Você escolheu: %s (ID=%d)\n", chosenServer.Name, chosenServer.ID)

	//Conectando ao nats do servidor escolhido pelo cliente
	nc, err := nats.Connect(chosenServer.NATS)
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS do servidor escolhido: %v", err)
	}
	defer nc.Close()
	log.Println("Conectado ao NATS do servidor escolhido: ", chosenServer.NATS)

	//Monta payload da escolha do servidor
	payload := map[string]int{"server_id": chosenServer.ID}

	payloadJSON, _ := json.Marshal(payload)

	//Cria Request
	req := shared.Request{
		ClientID: "cliente1",
		Action:   "CHOOSE_SERVER",
		Payload:  json.RawMessage(payloadJSON),
	}

	dataReq, _ := json.Marshal(req)

	//Publica no tópico do servidor escolhido
	topic := fmt.Sprintf("server.%d.requests", chosenServer.ID)
	err = nc.Publish(topic, dataReq)
	if err != nil {
		log.Printf("Erro ao publicar escolha do servidor: %v", err)
	} else {
		log.Printf("Escolha do servidor enviada com sucesso!")
	}

	for {
		option := utils.MenuInicial()

		switch option {
		case "1": // Cadastro
			data := utils.Cadastro()
			jsonData, err := json.Marshal(data)

			log.Printf("data: %s", data)
			log.Printf("\njson data: %s", jsonData)
			
			if err != nil {
    			log.Printf("Erro ao converter para JSON: %v", err)
    			return
			}
			req := shared.Request{
				ClientID: "cliente1",
				Action:   "REGISTER",
				Payload:  json.RawMessage(jsonData),
			}

			reqData, _ := json.Marshal(req)

			// Publica no tópico do servidor escolhido
			topic := fmt.Sprintf("server.%d.requests", chosenServer.ID)
			err = nc.Publish(topic, reqData)
			if err != nil {
				log.Printf("Erro ao publicar cadastro: %v", err)
			} else {
				log.Printf("Cadastro enviado com sucesso!")
			}

		case "3": // Sair
			return
		}
	}
}
