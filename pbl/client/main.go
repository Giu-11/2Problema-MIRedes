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
	fmt.Printf("Você escolheu: %s (ID=%d)\n", chosenServer.Name, chosenServer.ID)

	//Conexão NATS
	nc, err := nats.Connect(chosenServer.NATS)
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS do servidor escolhido: %v", err)
	}
	defer nc.Close()
	log.Println("Conectado ao NATS do servidor escolhido:", chosenServer.NATS)

	//ID do cliente para NATS
	clientID := fmt.Sprintf("cliente%d", utils.GerarIdAleatorio())
	log.Printf("Seu ID de cliente para esta sessão é: %s\n", clientID)

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
		log.Printf("Erro ao converter para JSON: %v", err)
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
		log.Println("Erro ao enviar requisição de login:", err)
		return shared.User{}, false
	}

	var response shared.Response
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		log.Printf("Erro ao decodificar resposta do servidor: %v", err)
		return shared.User{}, false
	}

	if response.Status == "success" {
		var user shared.User
		user.UserId = clientID
		if err := json.Unmarshal(response.Data, &user); err != nil {
			log.Printf("Erro ao decodificar dados do usuário: %v", err)
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
	//Inicia heartbeat
	serverTopic := fmt.Sprintf("server.%d.requests", server.ID)
	startHeartbeat(nc, clientID, serverTopic)

	//Inicia listener de mensagens do cliente 
	game.StartGameListener(nc, clientID)

	for {
		option := utils.ShowMenuPrincipal()
		switch option {
		case "1": //Entrar na fila
			fmt.Println("Entrando na fila para uma nova partida...")

			clientTopic := fmt.Sprintf("client.%s.inbox", clientID)
			success := game.JoinQueue(nc, server, &user, clientTopic)
			if !success {
				fmt.Println("Não foi possível entrar na fila.")
				continue
			}

			fmt.Println("Esperando por um adversário...")

			//Espera MATCH
			matchChan := make(chan shared.User, 1)
			doneChan := make(chan struct{}, 1)
			go waitForMatch(nc, clientTopic, matchChan, doneChan)

			opponent, ok := <-matchChan
			if !ok {
				fmt.Println("Falha ao iniciar a partida.")
				continue
			}
			
			fmt.Printf("\nPartida iniciada! Oponente: %s\n", opponent.UserId)
			opponentID := opponent.UserId

			//Loop de turnos
			for {
				card, ok := game.ChooseCard(user)
				if !ok {
					fmt.Println("Saindo da partida...")
					break
				}

				game.SendCardPlay(nc, server.ID, user.UserId, opponentID, card)

				//alternar turnos --> implementar ainda
			}

		case "2": //Deck
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

		case "5": //Regras
			utils.ShowRules()

		case "7": //Logout
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
		log.Printf("Erro ao decodificar resposta de logout: %v", err)
		return
	}

	if response.Status == "success" {
		log.Printf("Logout realizado com sucesso no servidor %d.", response.Server)
	} else {
		log.Printf("Falha no logout: %s", response.Error)
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


func waitForMatch(nc *nats.Conn, clientTopic string, matchChan chan<- shared.User, doneChan chan struct{}) {
	sub, err := nc.SubscribeSync(clientTopic)
	if err != nil {
		log.Printf("Erro ao subscrever no tópico do cliente: %v", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		msg, err := sub.NextMsg(1 * time.Second)
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			log.Printf("Erro ao receber mensagem NATS: %v", err)
			return
		}

		var resp shared.Response
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			log.Println("Erro ao decodificar mensagem do servidor:", err)
			continue
		}

		if resp.Action == "MATCH" {
			var opponent shared.User
			if err := json.Unmarshal(resp.Data, &opponent); err != nil {
				log.Println("Erro ao decodificar dados do adversário:", err)
				continue
			}

			select {
			case matchChan <- opponent:
			default:
			}
			close(doneChan)
			return
		}
	}
}
