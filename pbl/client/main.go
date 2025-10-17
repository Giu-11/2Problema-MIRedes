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
	servers := []ServerInfo{
		{ID: 1, Name: "Servidor 1", NATS: "nats://localhost:4223"},
		{ID: 2, Name: "Servidor 2", NATS: "nats://localhost:4224"},
		{ID: 3, Name: "Servidor 3", NATS: "nats://localhost:4225"},
	}

	chooseString := utils.EscolherServidor()
	chooseInt, err := strconv.Atoi(chooseString)
	if err != nil || chooseInt < 1 || chooseInt > len(servers) {
		fmt.Println("Escolha inválida.")
		return
	}

	chosenServer := servers[chooseInt-1]
	fmt.Printf("Você escolheu: %s (ID=%d)\n", chosenServer.Name, chosenServer.ID)

	nc, err := nats.Connect(chosenServer.NATS)
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS do servidor escolhido: %v", err)
	}
	defer nc.Close()
	log.Println("Conectado ao NATS do servidor escolhido:", chosenServer.NATS)

	clienteID := fmt.Sprintf("cliente%d", utils.GerarIdAleatorio())
	log.Printf("Seu ID de cliente para esta sessão é: %s\n", clienteID)

	// Lógica principal do menu
	handleMainMenu(nc, chosenServer, clienteID)
}

// handleMainMenu gerencia o loop do menu inicial (antes do login)
func handleMainMenu(nc *nats.Conn, server ServerInfo, clientID string) {
	for {
		option := utils.MenuInicial()
		switch option {
		case "1": // Login
			// A função de login agora retorna 'true' se o login for bem-sucedido
			success := handleLogin(nc, server, clientID)
			if success {
				// Se o login funcionou, entramos no menu do jogo
				handleGameMenu(nc, server, clientID)
			}
		case "2": // Sair
			fmt.Println("Até mais!")
			return
		default:
			fmt.Println("Opção inválida, tente novamente.")
		}
	}
}

// handleLogin cuida do processo de login e trata a resposta
func handleLogin(nc *nats.Conn, server ServerInfo, clientID string) bool {
	credentials := utils.Login()
	jsonData, err := json.Marshal(credentials)
	if err != nil {
		log.Printf("Erro ao converter para JSON: %v", err)
		return false
	}

	req := shared.Request{
		ClientID: clientID,
		Action:   "LOGIN",
		Payload:  json.RawMessage(jsonData),
	}
	reqData, _ := json.Marshal(req)

	topic := fmt.Sprintf("server.%d.requests", server.ID)
	msg, err := nc.Request(topic, reqData, 5*time.Second)
	if err != nil {
		if err == nats.ErrTimeout {
			log.Println("Erro: O servidor não respondeu a tempo.")
		} else {
			log.Printf("Erro ao enviar requisição de login: %v", err)
		}
		return false
	}

	var response shared.Response
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		log.Printf("Erro ao decodificar resposta do servidor: %v", err)
		return false
	}

	if response.Status == "success" {
		fmt.Println("\nLogin realizado com sucesso!")
		return true
	} else {
		fmt.Println("\nFalha no login:", response.Error)
		return false
	}
}

// handleGameMenu gerencia o loop do menu do jogo (após o login)
func handleGameMenu(nc *nats.Conn, server ServerInfo, clientID string) {
	// argumento não usado(nc) sera usado no futuro
	for {
		option := utils.ShowMenuPrincipal()
		switch option {
		case "1": //entrar na fila
			fmt.Println("Entrando na fila para uma nova partida...")
			// Aqui viria a lógica para enviar a requisição de matchmaking
		case "2": //ver e organizar deck
			fmt.Println("ver deck não implementado..")
		case "3": //abrir pacote
			fmt.Println("abrir pacote não implementado")
		case "4": // troca de cartas
			fmt.Println("troca de cartas não implentada")
		case "5": //ver regras
			utils.ShowRules()
		case "6": // ping
			fmt.Println("ping não implemntado")
		case "7": //sair
			fmt.Println("Deslogando...")
			return // Retorna para o menu principal
		default:
			fmt.Println("Opção ainda não implementada.")
		}
	}
}
