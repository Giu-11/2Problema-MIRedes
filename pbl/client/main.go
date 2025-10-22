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
	"pbl/style"

	"github.com/nats-io/nats.go"
)

func main() {
	servers := []models.ServerInfo{
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
	fmt.Printf("\nVocê escolheu: %s (ID=%d)\n", chosenServer.Name, chosenServer.ID)

	nc, err := nats.Connect(chosenServer.NATS)
	if err != nil {
		log.Fatalf("Erro ao conectar no NATS do servidor escolhido: %v", err)
	}
	defer nc.Close()
	fmt.Println("Conectado ao NATS do servidor escolhido:", chosenServer.NATS)

	clientID := fmt.Sprintf("cliente%d", utils.GerarIdAleatorio())
	fmt.Printf("\nSeu ID desta sessão é: %s\n", clientID)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		logout(nc, chosenServer, clientID)
		os.Exit(0)
	}()

	handleMainMenu(nc, chosenServer, clientID)
}

func handleMainMenu(nc *nats.Conn, server models.ServerInfo, clientID string) {
	for {
		option := utils.MenuInicial()
		switch option {
		case "1":
			user, success := sendLoginRequest(nc, server, clientID)
			if success {
				user.UserId = clientID
				startGameLoop(nc, server, clientID, user)
			}
		case "2":
			fmt.Println("Até mais!")
			return
		default:
			fmt.Println("Opção inválida, tente novamente.")
		}
	}
}

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

func startGameLoop(nc *nats.Conn, server models.ServerInfo, clientID string, user shared.User) {
	serverTopic := fmt.Sprintf("server.%d.requests", server.ID)
	startHeartbeat(nc, clientID, serverTopic)
	
	for {
		option := utils.ShowMenuPrincipal()
		switch option {
		case "1":
			clientTopic := fmt.Sprintf("client.%s.inbox", clientID)
			matchChan := make(chan game.MatchInfo, 1)
			
			sub := game.StartGameListener(nc, clientID, matchChan, user)
			
			success := game.JoinQueue(nc, server, &user, clientTopic)
			if !success {
				fmt.Println("Não foi possível entrar na fila.")
				if sub != nil {
					sub.Unsubscribe()
				}
				continue
			}

			fmt.Println("Esperando por um adversário...")
			
			select {
			case matchInfo := <-matchChan:
				if sub != nil {
					sub.Unsubscribe()
				}
				
				fmt.Printf("\n✓ Match confirmado! Jogando contra: %s\n", matchInfo.Opponent.UserName)
				playGame(nc, &matchInfo.Room, user, matchInfo.Opponent)
			}

		case "2":
			style.Clear()
			menuCard(nc, server, clientID, user)
		case "3":
			style.Clear()
			handleClientDrawCard(nc, server, clientID)
		case "4":
			style.Clear()
			fmt.Println("Troca de cartas não implementada")
		case "5":
			style.Clear()
			utils.ShowRules()
		case "6":
			style.Clear()
			fmt.Println("Ping não implementado")
		case "7":
			style.Clear()
			fmt.Println("Deslogando...")
			logout(nc, server, clientID)
			return

		default:
			fmt.Println("Opção inválida.")
		}
	}
}

func playGame(nc *nats.Conn, room *shared.GameRoom, currentUser shared.User, opponent shared.User) {
	fmt.Println("\nPARTIDA INICIADA")

	isMyTurn := room.Turn == currentUser.UserId
	if isMyTurn {
		fmt.Println("✓ Você começa!")
	} else {
		fmt.Printf("%s começa. Aguarde...\n", opponent.UserName)
	}

	gameOver := false
	alreadyPlayed := false // faz o controle das jogadas
	gameMsgChan := make(chan shared.GameMessage, 10)

	clientTopic := fmt.Sprintf("client.%s.inbox", currentUser.UserId)
	sub, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		var gameMsg shared.GameMessage
		if err := json.Unmarshal(msg.Data, &gameMsg); err != nil {
			log.Println("Erro ao decodificar mensagem:", err)
			return
		}
		gameMsgChan <- gameMsg
	})
	if err != nil {
		log.Println("Erro ao criar subscription:", err)
		return
	}
	defer sub.Unsubscribe()

	if isMyTurn {
		card, ok := game.ChooseCard(currentUser)
		if !ok {
			fmt.Println("Você desistiu da partida.")
			return
		}
		game.SendCardPlay(nc, room, currentUser.UserId, card)
		fmt.Printf("\nVocê jogou: %s (%s)\n", card.Element, card.Type)
		fmt.Println("Aguardando adversário...")
		alreadyPlayed = true //já jogou
	}

	for !gameOver {
		select {
		case gameMsg := <-gameMsgChan:
			switch gameMsg.Type {
			case "PLAY_CARD":
				//Mensagem do adversário jogando
				if gameMsg.From == opponent.UserId {
					var card shared.Card
					if len(gameMsg.Data) > 0 {
						if err := json.Unmarshal(gameMsg.Data, &card); err != nil {
							log.Println("Erro ao decodificar carta:", err)
							continue
						}
					}

					fmt.Printf("\n%s jogou: %s (%s)\n", opponent.UserName, card.Element, card.Type)

					if !alreadyPlayed {
						room.Turn = currentUser.UserId
						fmt.Println("\nSua vez!")
						
						chosenCard, ok := game.ChooseCard(currentUser)
						if ok {
							game.SendCardPlay(nc, room, currentUser.UserId, chosenCard)
							fmt.Printf("Você jogou: %s (%s)\n", chosenCard.Element, chosenCard.Type)
							fmt.Println("Aguardando resultado...")
							alreadyPlayed = true //cliente jogou
						} else {
							fmt.Println("Você desistiu da partida.")
							return
						}
					} else {
						fmt.Println("Aguardando resultado da rodada...")
					}
				}

			case "ROUND_RESULT":
				fmt.Println("\n--------------------------------")
				fmt.Println("            Resultado           ")
				fmt.Println("--------------------------------")
				
				if gameMsg.Winner == nil {
					fmt.Println("Empate dos jogadores!")
				} else {
					fmt.Println("Vencedor: ", gameMsg.Winner.UserName)
				}

				alreadyPlayed = false
				gameOver = true

			case "GAME_OVER":
				fmt.Println("\nJogo finalizado!")
				gameOver = true
			}

		case <-time.After(90 * time.Second):
			fmt.Println("\nTimeout: servidor não respondeu.")
			gameOver = true
		}
	}

	fmt.Print("Pressione ENTER para voltar ao menu principal...")
	fmt.Scanln()
	style.Clear()
}

func menuCard(nc *nats.Conn, server models.ServerInfo, clientID string, user shared.User){
	sair := false
	for !sair{
		option := utils.ShowMenuCards()
		switch option{
		case "1":
			style.Clear()
			handleClientSeeDeck(nc, server, clientID)
		case "2":
			handleChangeDeck(nc, server, clientID)
		case "3":
			sair = true
		}
	}
}

// logout envia mensagem de LOGOUT
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

func handleClientDrawCard(nc *nats.Conn, server models.ServerInfo, clienteID string) {
	fmt.Println("Enviando requisição para pegar uma carta...")
	req := shared.Request{
		ClientID: clienteID,
		Action:   "OPEN_PACK",
		Payload:  nil,
	}
	reqData, _ := json.Marshal(req)

	topic := fmt.Sprintf("server.%d.requests", server.ID)
	msg, err := nc.Request(topic, reqData, 5*time.Second)

	if err != nil {
		log.Printf("Erro na requisição para pegar carta: %v", err)
		return // Sai da função imediatamente para evitar o crash.
	}

	// Como segurança extra, verificamos se a mensagem é válida antes de usá-la.
	if msg == nil || msg.Data == nil {
		log.Printf("O servidor retornou uma resposta vazia.")
		return
	}

	var response shared.Response
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		log.Printf("Erro ao decodificar resposta da carta: %v", err)
		return
	}

	if response.Status == "success" {
		var drawnData shared.CardDrawnData
		if err := json.Unmarshal(response.Data, &drawnData); err != nil {
			log.Printf("Erro ao decodificar os dados da carta: %v", err)
			return
		}
		style.PrintVerd("\n[SUCESSO] Você pegou uma carta!\n")
		fmt.Printf("   -> Carta: %s %s\n", drawnData.Card.Element, drawnData.Card.Type)
	} else {
		msg := fmt.Sprintf("\n[FALHA] Não foi possível pegar a carta: %s\n", response.Error)
		style.PrintVerm(msg)
	}
}

func handleClientSeeDeck(nc *nats.Conn, server models.ServerInfo, clientID string)[]shared.Card{
	fmt.Println("Buscando cartas...")
	req := shared.Request{
		ClientID: clientID,
		Action: "SEE_CARDS",
		Payload: nil,
	}
	reqData,_ := json.Marshal(req)

	topic := fmt.Sprintf("server.%d.requests", server.ID)
	msg, err := nc.Request(topic, reqData, 5*time.Second)

	if err != nil {
		log.Printf("Erro na requisição para pegar carta: %v", err)
		return nil// Sai da função imediatamente para evitar o crash
	}

	if msg == nil || msg.Data == nil{
		log.Printf("O servidor retornou uma resposta vazia.")
		return nil
	}

	var response shared.Response
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		log.Printf("Erro ao decodificar resposta do inventário: %v", err)
		return nil
	}

	if response.Status == "success"{
		var inventario shared.Cards
		if err := json.Unmarshal(response.Data, &inventario); err != nil {
			log.Printf("Erro ao decodificar os dados do inventario: %v", err)
			return nil
		}
		utils.MostrarInventario(inventario.Cards)
		return inventario.Cards
	} else {
		msg := fmt.Sprintf("\n[FALHA] Não foi possível ver inventário: %s\n", response.Error)
		style.PrintVerm(msg)
	}

	return nil
}

func handleChangeDeck(nc *nats.Conn, server models.ServerInfo, clientID string){
	cards := handleClientSeeDeck(nc, server, clientID)
	var selectedCards []int
	var deck []shared.Card
	if cards != nil{
		for i := range(4){
			valida := false
			for ! valida{
				fmt.Printf("Digite o número da %d° carta para o baralho: ", i+1)
				in := utils.ReadLineSafe()
				if inInt, err := strconv.Atoi(in); err == nil {
					if inInt>=0 && inInt<len(cards) && !utils.Contains(selectedCards, inInt){
						deck = append(deck, cards[inInt])
						selectedCards = append(selectedCards, inInt)
						valida = true
					}else{
						style.PrintMag("Valor inválido\n")
					}
				}else{
					style.PrintMag("Valor inválido, digite o número da carta!\n")
				}
			}
		}
		fmt.Println(deck)
	}

	//TODO: Separar ^ em uma função
	//TODO: mandar deck novo ao servidor
	//TODO: mostrar deck montado ao cliente
}

// startHeartbeat envia HEARTBEAT periódico
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