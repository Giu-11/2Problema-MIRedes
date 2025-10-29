package main

import (
	"encoding/json"
	"fmt"
	"log"
	"pbl/shared"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// --- Variáveis de Configuração do Teste ---

const (
	// NATS do server 1
	testNatsURL = "nats://localhost:4223"
	// tópico do server 1
	testServerTopic = "server.1.requests"
	// Timeout padrão para respostas
	testTimeout = 15 * time.Second 
)

// --- Helper: O Cliente Falso ---

// cliente falso
type TestClient struct {
	t        *testing.T       // Referência ao helper de teste
	nc       *nats.Conn       // Conexão NATS real
	clientID string           // ID único para este cliente
	inbox    chan *nats.Msg   // Canal para onde o NATS envia mensagens (MATCH, etc)
	sub      *nats.Subscription // A inscrição NATS
	user     shared.User      // Dados do usuário após o login
}

// cria e conecta um novo cliente falso.
func newTestClient(t *testing.T) *TestClient {
	opts := []nats.Option{
		nats.ReconnectBufSize(5 * 1024 * 1024),
	}
	nc, err := nats.Connect(testNatsURL, opts...)
	if err != nil {
		t.Errorf("Falha ao conectar ao NATS em %s: %v. O servidor NATS está rodando?", testNatsURL, err)
		return nil
	}

	clientID := "testclient-" + uuid.New().String()[:8]
	inboxChan := make(chan *nats.Msg, 10) // Buffer maior
	clientTopic := fmt.Sprintf("client.%s.inbox", clientID)

	sub, err := nc.Subscribe(clientTopic, func(msg *nats.Msg) {
		inboxChan <- msg
	})
	if err != nil {
		t.Errorf("Falha ao se inscrever no tópico de inbox %s: %v", clientTopic, err)
		nc.Close()
		return nil
	}

	return &TestClient{
		t:        t,
		nc:       nc,
		clientID: clientID,
		inbox:    inboxChan,
		sub:      sub,
	}
}


func (c *TestClient) logout() {
	if c.user.UserName == "" {
		return
	}

	req := shared.Request{
		ClientID: c.clientID,
		Action:   "LOGOUT",
	}
	reqData, _ := json.Marshal(req)

	_, err := c.nc.Request(testServerTopic, reqData, 2*time.Second)
	if err != nil {
		c.t.Logf("[%s] Aviso: Erro na requisição de LOGOUT durante o Close(): %v", c.clientID, err)
	}
}


// desconecta o cliente.
func (c *TestClient) Close() {
	c.logout()
	if c.sub != nil {
		c.sub.Unsubscribe()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	if c.inbox != nil {
		close(c.inbox)
	}
}

// espera por uma mensagem assíncrona no inbox do cliente.
func (c *TestClient) waitForMessage(timeout time.Duration) (*nats.Msg, error) {
	select {
	case msg, ok := <-c.inbox:
		if !ok {
			return nil, fmt.Errorf("canal de inbox fechado")
		}
		return msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout esperando mensagem no inbox %s", c.sub.Subject)
	}
}

// --- Ações do Cliente Falso ---

// simula um cliente fazendo login.
// Retorna 'true' em sucesso, 'false' em falha.
func (c *TestClient) login(username string) bool {
	creds := shared.User{UserName: username, Password: "123"}
	payload, _ := json.Marshal(creds)
	req := shared.Request{
		ClientID: c.clientID,
		Action:   "LOGIN",
		Payload:  payload,
	}
	reqData, _ := json.Marshal(req)

	msg, err := c.nc.Request(testServerTopic, reqData, testTimeout)
	if err != nil {
		c.t.Errorf("[%s] Erro na requisição de LOGIN: %v", c.clientID, err)
		return false
	}

	var resp shared.Response
	json.Unmarshal(msg.Data, &resp)
	if resp.Status != "success" {
		c.t.Errorf("[%s] Falha no LOGIN: %s", c.clientID, resp.Error)
		return false
	}

	json.Unmarshal(resp.Data, &c.user)
	c.user.UserId = c.clientID
	c.user.ServerID = 1 // Hardcoded para o server 1
	// c.t.Logf("[%s] Login como '%s' bem-sucedido", c.clientID, c.user.UserName)
	return true
}

// simula um cliente abrindo um pacote.
// Retorna 'true' em sucesso, 'false' em falha.
func (c *TestClient) openPack() bool {
	req := shared.Request{
		ClientID: c.clientID,
		Action:   "OPEN_PACK",
	}
	reqData, _ := json.Marshal(req)

	msg, err := c.nc.Request(testServerTopic, reqData, testTimeout)
	if err != nil {
		c.t.Errorf("[%s] Erro na requisição OPEN_PACK: %v", c.clientID, err)
		return false
	}

	var resp shared.Response
	json.Unmarshal(msg.Data, &resp)
	if resp.Status != "success" {
		c.t.Errorf("[%s] Falha no OPEN_PACK: %s", c.clientID, resp.Error)
		return false
	}

	var cardData shared.CardDrawnData
	if err := json.Unmarshal(resp.Data, &cardData); err != nil {
		c.t.Errorf("[%s] Falha ao decodificar dados da carta: %v", c.clientID, err)
		return false
	}

	if cardData.Card.Element == "" {
		c.t.Errorf("[%s] Recebida uma carta vazia", c.clientID)
		return false
	}
	// c.t.Logf("[%s] Sucesso! Carta recebida: %s %s", c.clientID, cardData.Card.Element, cardData.Card.Type)
	return true
}

// simula um cliente entrando na fila.
// Retorna 'true' em sucesso, 'false' em falha.
func (c *TestClient) joinQueue() bool {
	c.user.Status = "available" 

    entry := shared.QueueEntry{
        Player:   c.user,
        ServerID: fmt.Sprintf("%d", c.user.ServerID),
        Topic:    c.sub.Subject,
        JoinTime: time.Now(),
    }
	payload, _ := json.Marshal(entry)
	req := shared.Request{
		Action:   "JOIN_QUEUE",
		ClientID: c.clientID,
		Payload:  json.RawMessage(payload),
	}
	reqData, _ := json.Marshal(req)

	msg, err := c.nc.Request(testServerTopic, reqData, testTimeout)
	if err != nil {
		c.t.Errorf("[%s] Erro na requisição JOIN_QUEUE: %v", c.clientID, err)
		return false
	}

	var resp shared.Response
	json.Unmarshal(msg.Data, &resp)
	if resp.Status != "success" {
		c.t.Errorf("[%s] Falha no JOIN_QUEUE: %s", c.clientID, resp.Error)
		return false
	}
	// c.t.Logf("[%s] Na fila. Aguardando partida...", c.clientID)
	return true
}

// --- OS TESTES ---

// testa o fluxo de login e abertura de pacote (1 CLIENTE)
func TestIntegration_OpenPack(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração (requer servidor rodando)")
	}

	client := newTestClient(t)
	if client == nil { t.Fatal("Falha ao criar cliente") }
	defer client.Close()

	if !client.login("testUser_Pack") {
		t.Fatal("Login falhou")
	}
	
	if !client.openPack() {
		t.Fatal("OpenPack falhou")
	}
}

// testa o fluxo de login e matchmaking local (2 CLIENTES)
func TestIntegration_Matchmaking(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração (requer servidor rodando)")
	}

	// Cria dois clientes
	clientA := newTestClient(t)
	if clientA == nil { t.Fatal("Falha ao criar cliente A") }
	defer clientA.Close()

	clientB := newTestClient(t)
	if clientB == nil { t.Fatal("Falha ao criar cliente B") }
	defer clientB.Close()

	// Login
	var wgLogin sync.WaitGroup
	wgLogin.Add(2)
	go func() { defer wgLogin.Done(); clientA.login("testUser_MatchA") }()
	go func() { defer wgLogin.Done(); clientB.login("testUser_MatchB") }()
	wgLogin.Wait()

	// Entra na fila
	var wgQueue sync.WaitGroup
	wgQueue.Add(2)
	go func() { defer wgQueue.Done(); clientA.joinQueue() }()
	go func() { defer wgQueue.Done(); clientB.joinQueue() }()
	wgQueue.Wait()

	// Espera pela partida
	t.Log("Clientes A e B na fila. Aguardando mensagem 'MATCH'...")
	var wgMatch sync.WaitGroup
	wgMatch.Add(2)
	var roomA, roomB shared.GameRoom

	// Listener do Cliente A
	go func() {
		defer wgMatch.Done()
		timeout := time.After(testTimeout)
		for {
			select {
			case msg, ok := <-clientA.inbox:
				if !ok { return } // Canal fechado
				var resp shared.Response
				if err := json.Unmarshal(msg.Data, &resp); err == nil && resp.Action == "MATCH" {
					// Encontrou o MATCH!
					json.Unmarshal(resp.Data, &roomA)
					t.Logf("[Cliente A] Partida recebida! Sala: %s", roomA.ID)
					return // Sucesso
				}
				// Se não for MATCH, ignora e continua
			case <-timeout:
				clientA.t.Errorf("Cliente A deu timeout esperando pelo MATCH")
				return
			}
		}
	}()

	// Listener do Cliente B
	go func() {
		defer wgMatch.Done()
		timeout := time.After(testTimeout)
		for {
			select {
			case msg, ok := <-clientB.inbox:
				if !ok { return } // Canal fechado
				var resp shared.Response
				if err := json.Unmarshal(msg.Data, &resp); err == nil && resp.Action == "MATCH" {
					// Encontrou o MATCH!
					json.Unmarshal(resp.Data, &roomB)
					t.Logf("[Cliente B] Partida recebida! Sala: %s", roomB.ID)
					return // Sucesso
				}
				// Se não for MATCH, ignora e continua
			case <-timeout:
				clientB.t.Errorf("Cliente B deu timeout esperando pelo MATCH")
				return
			}
		}
	}()

	wgMatch.Wait()

	// Verificação Final
	if roomA.ID == "" || roomB.ID == "" { t.Fatal("Falha ao receber dados da sala") }
	if roomA.ID != roomB.ID { t.Errorf("IDs da sala não batem! A: %s, B: %s", roomA.ID, roomB.ID) }
	t.Logf("Sucesso! Partida %s criada para A e B.", roomA.ID)
}

// simula N clientes pedindo cartas ao mesmo tempo
func TestIntegration_StressOpenPack(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração (requer servidor rodando)")
	}
	t.Parallel() // Permite rodar em paralelo com outros testes

	const CLIENT_COUNT = 50 // 50 clientes simultâneos
	log.Printf("[STRESS_PACK] Iniciando teste com %d clientes...", CLIENT_COUNT)

	var wg sync.WaitGroup
	var successCounter int32

	wg.Add(CLIENT_COUNT)

	for i := 0; i < CLIENT_COUNT; i++ {
		go func(i int) {
			defer wg.Done()
			
			client := newTestClient(t)
			if client == nil {
				return 
			}
			defer client.Close()

			username := fmt.Sprintf("stressUser_Pack_%d", i)
			
			if !client.login(username) {
				return 
			}
			
			if !client.openPack() {
				return 
			}
			
			atomic.AddInt32(&successCounter, 1)

		}(i)
	}

	// Espera todos os clientes terminarem
	wg.Wait()

	// Verificação Final
	finalSuccesses := atomic.LoadInt32(&successCounter)
	log.Printf("[STRESS_PACK] Teste concluído: %d/%d clientes pegaram uma carta com sucesso.", finalSuccesses, CLIENT_COUNT)
	
	if finalSuccesses != CLIENT_COUNT {
		t.Fatalf("Teste de stress falhou! Esperado %d sucessos, mas obteve %d", CLIENT_COUNT, finalSuccesses)
	}
}


// simula N clientes entrando na fila ao mesmo tempo
func TestIntegration_StressMatchmaking(t *testing.T) {
	if testing.Short() {
		t.Skip("Pulando teste de integração (requer servidor rodando)")
	}
	t.Parallel() // Permite rodar em paralelo com outros testes

	const CLIENT_COUNT = 50 
	log.Printf("[STRESS_MATCH] Iniciando teste com %d clientes...", CLIENT_COUNT)

	var wg sync.WaitGroup
	var successCounter int32 

	wg.Add(CLIENT_COUNT)

	for i := 0; i < CLIENT_COUNT; i++ {
		go func(i int) {
			defer wg.Done()
			
			client := newTestClient(t)
			if client == nil { return }
			defer client.Close()

			username := fmt.Sprintf("stressUser_Match_%d", i)
			
			if !client.login(username) { return }
			if !client.joinQueue() { return }

			// Listener do cliente 
			timeout := time.After(30 * time.Second) 
			for {
				select {
				case msg, ok := <-client.inbox:
					if !ok { return } // Canal fechado
					var resp shared.Response
					if err := json.Unmarshal(msg.Data, &resp); err == nil && resp.Action == "MATCH" {
						// Encontrou o MATCH!
						atomic.AddInt32(&successCounter, 1)
						return // Sucesso
					}
					// Se não for MATCH, ignora e continua
				case <-timeout:
					client.t.Errorf("[%s] deu timeout esperando pela partida MATCH", client.clientID)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verificação Final
	finalSuccesses := atomic.LoadInt32(&successCounter)
	log.Printf("[STRESS_MATCH] Teste concluído: %d/%d clientes receberam uma partida.", finalSuccesses, CLIENT_COUNT)
	
	if finalSuccesses != CLIENT_COUNT {
		t.Fatalf("Teste de stress de matchmaking falhou! Esperado %d matches, mas obteve %d", CLIENT_COUNT, finalSuccesses)
	}
}