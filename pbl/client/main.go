package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

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

	cliente := fmt.Sprintf("cliente%d", utils.GerarIdAleatorio())

	//Cria Request
	req := shared.Request{
		ClientID: cliente,
		Action:   "CHOOSE_SERVER",
		Payload:  json.RawMessage(payloadJSON),
	}

	dataReq, _ := json.Marshal(req)

	//Publica no tópico do servidor escolhido
	topic := fmt.Sprintf("server.%d.requests", chosenServer.ID)
	msg, err := nc.Request(topic, dataReq, 3*time.Second)
	if err != nil {
		log.Fatalf("Erro ao publicar escolha do servidor: %v", err)
	}

	log.Println("Resposta do servidor: ", string(msg.Data))

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
				ClientID: cliente,
				Action:   "REGISTER",
				Payload:  json.RawMessage(jsonData),
			}

			reqData, _ := json.Marshal(req)

			topic := fmt.Sprintf("server.%d.requests", chosenServer.ID)
			msg, err := nc.Request(topic, reqData, 5*time.Second)
			if err != nil {
				if err == nats.ErrTimeout {
					log.Printf("Timeout ao aguardar resposta do servidor. Tentando novamente...")
					// Opcional: adicionar retry
					return
				}
				log.Printf("Erro ao publicar cadastro: %v", err)
				return
			}

			// Verificar se a mensagem não é nil antes de usar
			if msg == nil {
				log.Printf("Resposta vazia do servidor")
				return
			}

			// Agora é seguro usar msg.Data
			var response shared.Response
			if err := json.Unmarshal(msg.Data, &response); err != nil {
				log.Printf("Erro ao decodificar resposta: %v", err)
				return
			}

			log.Println("Resposta do servidor:", string(msg.Data))

		case "2": //Login
			data := utils.Login()
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("Erro ao converter para JSON: %v", err)
				return
			}
			req := shared.Request{
				ClientID: cliente,
				Action:   "LOGIN",
				Payload:  json.RawMessage(jsonData),
			}

			reqData, _ := json.Marshal(req)

			topic := fmt.Sprintf("server.%d.requests", chosenServer.ID)
			msg, err := nc.Request(topic, reqData, 5*time.Second)
			if err != nil {
				log.Printf("Erro ao publicar cadastro: %v", err)
			} else {
				utils.ShowMenuLogin()
			}
			log.Println("Resposta do servidor:", string(msg.Data))

		case "3": // Sair
			return
		}
	}
}
