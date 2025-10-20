package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"pbl/client/game"
	"pbl/client/models"
	"pbl/client/utils"
	"pbl/shared"
	//"pbl/style"

	"github.com/nats-io/nats.go"
)

func main() {
	//Lista de servidores disponíveis
	servers := []models.ServerInfo{
		{ID: 1, Name: "Servidor 1", NATS: "nats://localhost:4223"},
		{ID: 2, Name: "Servidor 2", NATS: "nats://localhost:4224"},
		{ID: 3, Name: "Servidor 3", NATS: "nats://localhost:4225"},
	}

	//Escolha do servidor
	chooseString := utils.EscolherServidor()
	chooseInt, err := strconv.Atoi(chooseString)
	if err != nil || chooseInt < 1 || chooseInt > len(servers) {
		fmt.Println("Escolha inválida.")
		return
	}
	chosenServer := servers[chooseInt-1]
	fmt.Printf("\nVocê escolheu: %s (ID=%d)\n", chosenServer.Name, chosenServer.ID)

	//Conexão NATS
	nc, err := nats.Connect(chosenServer.NATS)
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS do servidor escolhido: %v", err)
	}
	defer nc.Close()
	fmt.Println("Conectado ao NATS do servidor escolhido:", chosenServer.NATS)

	//ID do cliente para NATS
	clientID := fmt.Sprintf("cliente%d", utils.GerarIdAleatorio())
	fmt.Printf("\nSeu ID desta sessão é: %s\n", clientID)

	//Captura Ctrl+C para logout
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		logout(nc, chosenServer, clientID)
		os.Exit(0)
	}()

	//Inicia menu inicial
	handleMainMenu(nc, chosenServer, clientID)
}

//Menu inicial
func handleMainMenu(nc *nats.Conn, server models.ServerInfo, clientID string) {
	for {
		option := utils.MenuInicial()
		switch option {
		case "1": //Login
			user, success := sendLoginRequest(nc, server, clientID)
			if success {
				user.UserId = clientID
				startGameLoop(nc, server, clientID, user)
			}
		case "2": //Sair
			fmt.Println("Até mais!")
			return
		default:
			fmt.Println("Opção inválida, tente novamente.")
		}
	}
}

//Login
func sendLoginRequest(nc *nats.Conn, server models.ServerInfo, clientID string) (shared.User, bool) {
	credentials := utils.Login()
	jsonData, err := json.Marshal(credentials)
	if err != nil {
		fmt.Printf("\nErro ao converter para JSON: %v", err)
		return shared.User{}, false
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
		fmt.Println("Erro ao enviar requisição de login:", err)
		return shared.User{}, false
	}

	var response shared.Response
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		fmt.Printf("\nErro ao decodificar resposta do servidor: %v", err)
		return shared.User{}, false
	}

	if response.Status == "success" {
		var user shared.User
		user.UserId = clientID
		if err := json.Unmarshal(response.Data, &user); err != nil {
			fmt.Printf("\nErro ao decodificar dados do usuário: %v", err)
			return shared.User{}, false
		}
		fmt.Println("\nLogin realizado com sucesso!")
		return user, true
	}

	fmt.Println("\nFalha no login:", response.Error)
	return shared.User{}, false
}

//Loop principal do jogo
func startGameLoop(nc *nats.Conn, server models.ServerInfo, clientID string, user shared.User) {
	serverTopic := fmt.Sprintf("server.%d.requests", server.ID)
	startHeartbeat(nc, clientID, serverTopic)
	for {
		option := utils.ShowMenuPrincipal()
		switch option {
		case "1": // Entrar na fila
			clientTopic := fmt.Sprintf("client.%s.inbox", clientID)
			
			// Cria canal para receber o match
			matchChan := make(chan shared.User, 1)
			
			// Inicia listener COM o canal
			game.StartGameListener(nc, clientID, matchChan)
			
			success := game.JoinQueue(nc, server, &user, clientTopic)
			if !success {
				fmt.Println("Não foi possível entrar na fila.")
				continue
			}

			fmt.Println("Esperando por um adversário...")
			
			// Aguarda match
			select {
			case opponent := <-matchChan:
				fmt.Printf("\nMatch confirmado! Jogando contra: %s\n", opponent.UserName)
				opponentID := opponent.UserId

				// Loop de turnos
				for {
					card, ok := game.ChooseCard(user)
					if !ok {
						fmt.Println("Saindo da partida...")
						break
					}
					game.SendCardPlay(nc, server.ID, user.UserId, opponentID, card)
				}
			}

		case "2": // Deck
			optionGame := utils.ShowMenuDeck()
			switch optionGame {
			case "1":
				fmt.Println("Visualizando todas as cartas do usuário...")
			case "2":
				fmt.Println("Visualizando cartas do deck...")
			case "3":
				fmt.Println("Alterar deck selecionado...")
			case "4":
				continue
			default:
				fmt.Println("Opção inválida.")
			}

		case "5": // Regras
			utils.ShowRules()

		case "7": // Logout
			fmt.Println("Deslogando...")
			logout(nc, server, clientID)
			return

		default:
			fmt.Println("Opção inválida.")
		}
	}
}
//Logout
func logout(nc *nats.Conn, server models.ServerInfo, clientID string) {
	req := shared.Request{
		ClientID: clientID,
		Action:   "LOGOUT",
	}
	reqData, _ := json.Marshal(req)

	topic := fmt.Sprintf("server.%d.requests", server.ID)
	msg, err := nc.Request(topic, reqData, 5*time.Second)
	if err != nil {
		log.Println("Erro ao enviar requisição de logout:", err)
		return
	}

	var response shared.Response
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		fmt.Printf("\nErro ao decodificar resposta de logout: %v", err)
		return
	}

	if response.Status == "success" {
		fmt.Printf("\nLogout realizado com sucesso no servidor %d.", response.Server)
	} else {
		fmt.Printf("\nFalha no logout: %s", response.Error)
	}
}

//Heartbeat
func startHeartbeat(nc *nats.Conn, clientID, serverTopic string) {
	go func() {
		for {
			req := shared.Request{
				Action:  "HEARTBEAT",
				Payload: nil,
			}
			data, _ := json.Marshal(req)
			nc.Publish(serverTopic, data)
			time.Sleep(5 * time.Second)
		}
	}()
}


