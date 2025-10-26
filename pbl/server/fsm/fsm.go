package fsm

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"sync"
	"time"

	"pbl/server/cards"

	sharedRaft "pbl/server/shared"
	"pbl/shared"

	"github.com/hashicorp/raft"
)

// maquina de estados finitos
type FSM struct {
	mu           sync.Mutex
	users        map[string]shared.User // mapa de usuarios
	cardStock    []shared.Card
	pendingCards map[string]shared.Card
	//Para a parte global
	GlobalQueue []shared.QueueEntry
	GlobalQueueMu sync.Mutex

	GlobalRooms map[string]*shared.GameRoom
	GlobalRoomsMu sync.RWMutex

	// cartas pendentes por sala
    PendingRoomCards map[string][]shared.Card
    PendingMu        sync.Mutex

	CreatedRooms chan *shared.GameRoom
	Raft *raft.Raft
}

func NewFSM() *FSM {
	return &FSM{
		users:        make(map[string]shared.User),
		cardStock:    cards.GerarEstoque(),
		pendingCards: make(map[string]shared.Card),
		GlobalRooms:  make(map[string]*shared.GameRoom),
		CreatedRooms: make(chan *shared.GameRoom, 10),
	}
}

// aplica um comando ao raft
func (fsm *FSM) Apply(logEntry *raft.Log) interface{} {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	var cmd sharedRaft.Command
	if err := json.Unmarshal(logEntry.Data, &cmd); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	switch cmd.Type {
	case sharedRaft.CommandOpenPack:
		var payload sharedRaft.DrawCardPayload
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal DrawCardPayload: %w", err)
		}

		if card, exists := fsm.pendingCards[payload.RequestID]; exists {
			log.Printf("[FSM] Comando DRAW_CARD repetido para RequestID %s. Retornando carta já pendente: %s", payload.RequestID, card.Type)
			return card
		}

		if len(fsm.cardStock) == 0 {
			log.Println("[FSM] Tentativa de pegar carta do estoque, mas está vazio.")
			fsm.cardStock = cards.GerarEstoque()
			log.Println("\tNOVAS CARTAS ADICIONADAS NO ESTOQUE")
			//return "STOCK_EMPTY"
		}

		drawnCard := fsm.cardStock[0]
		fsm.cardStock = fsm.cardStock[1:]
		fsm.pendingCards[payload.RequestID] = drawnCard

		log.Printf("[FSM] Carta '%s' reservada para RequestID %s. Estoque restante: %d", drawnCard.Element, payload.RequestID, len(fsm.cardStock))
		return drawnCard

	case sharedRaft.CommandClaimCard:
		var payload sharedRaft.ClaimCardPayload
		if err := json.Unmarshal(cmd.Data, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal ClaimCardPayload: %w", err)
		}

		delete(fsm.pendingCards, payload.RequestID)
		log.Printf("[FSM] Carta para RequestID %s foi reivindicada e removida de pendentes.", payload.RequestID)
		return nil

	case sharedRaft.CommandQueueJoinGlobal:
		var entry shared.QueueEntry
		if err := json.Unmarshal(cmd.Data, &entry); err != nil {
			return err
		}

		// Evita duplicatas
		exists := false
		fsm.GlobalQueueMu.Lock()
		for _, e := range fsm.GlobalQueue {
			if e.Player.UserId == entry.Player.UserId {
				exists = true
				break
			}
		}
		if exists {
			return nil
		}
		
		fsm.GlobalQueue = append(fsm.GlobalQueue, entry)
		fsm.GlobalQueueMu.Unlock()
		
		log.Printf("[FSM] Usuário %s adicionado à fila global", entry.Player.UserName)

		// APENAS líder cria partidas
		if fsm.Raft != nil && fsm.Raft.State() == raft.Leader {
			log.Println("[FSM] Sou o LÍDER! Tentando criar partidas...")
			go fsm.TryMatchPlayers()
		}

		return nil


	case sharedRaft.CommandCreateRoom:
		log.Println("Chamou o create")
		var room shared.GameRoom
		if err := json.Unmarshal(cmd.Data, &room); err != nil {
			log.Printf("[FSM] Erro ao criar sala: %v", err)
			return err
		}
		fsm.GlobalRoomsMu.Lock()
		fsm.GlobalRooms[room.ID] = &room
		fsm.GlobalRoomsMu.Unlock()
		log.Printf("[FSM] Sala criada: %s (%s vs %s)", room.ID, room.Player1.UserName, room.Player2.UserName)

		//notifica internamente (sem rede)
		select {
		case fsm.CreatedRooms <- &room:
		default:
			log.Println("[FSM] Aviso: fila de createdRooms cheia, descartando notificação.")
		}

		return nil


	case sharedRaft.CommandQueueLeave:
		var entry shared.QueueEntry
		if err := json.Unmarshal(cmd.Data, &entry); err != nil {
			log.Printf("[FSM] Erro ao decodificar LEAVE_QUEUE: %v", err)
			return err
		}
		fsm.GlobalQueueMu.Lock()
		for i, e := range fsm.GlobalQueue {
			if e.Player.UserId == entry.Player.UserId {
				fsm.GlobalQueue = append(fsm.GlobalQueue[:i], fsm.GlobalQueue[i+1:]...)
				log.Printf("[FSM] Usuário %s removido da fila global", entry.Player.UserName)
				break
			}
		}
		fsm.GlobalQueueMu.Unlock()
		return nil
	default:
		return fmt.Errorf("unrecognized command type: %s", cmd.Type)
	}
}

type FSMState struct {
	CardStock    []shared.Card
	PendingCards map[string]shared.Card
}

// cria saves do estado do servidor
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	state := &FSMState{
		CardStock:    make([]shared.Card, len(f.cardStock)),
		PendingCards: make(map[string]shared.Card),
	}
	copy(state.CardStock, f.cardStock)
	for k, v := range f.pendingCards {
		state.PendingCards[k] = v
	}

	return &fsmSnapshot{state: state}, nil
}

// Restaura a fsm a partir de um snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var state FSMState
	if err := json.NewDecoder(rc).Decode(&state); err != nil {
		return err
	}
	f.cardStock = state.CardStock
	f.pendingCards = state.PendingCards
	return nil
}

// snapshot do estado do servidor
type fsmSnapshot struct {
	state *FSMState
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		if err := json.NewEncoder(sink).Encode(s.state); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}
	return err
}

func (s *fsmSnapshot) Release() {
	// vazia que não tem nada que precisa ser limpado depois do snapshot

}

func (fsm *FSM) TryMatchPlayers() {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if fsm.Raft == nil || fsm.Raft.State() != raft.Leader {
		log.Println("[FSM] NÃO SOU O LÍDER, retornando")
		return
	}

	fsm.GlobalQueueMu.Lock()
	for len(fsm.GlobalQueue) >= 2 {
		player1 := fsm.GlobalQueue[0].Player
		player2 := fsm.GlobalQueue[1].Player
		fsm.GlobalQueue = fsm.GlobalQueue[2:]

		turn := chooseRandomPlayer(player1.UserId, player2.UserId)

		room := shared.GameRoom{
			ID:        fmt.Sprintf("global-%s-vs-%s", player1.UserName, player2.UserName),
			Player1:   &player1,
			Player2:   &player2,
			Turn:      turn,
			Status:    shared.WaitingPlayers,
			Server1ID: player1.ServerID,
			Server2ID: player2.ServerID,
		}

		// Escolhe host
		room.ServerID = chooseRandomPlayerInt(room.Server1ID, room.Server2ID)

		cmd := sharedRaft.Command{
			Type: sharedRaft.CommandCreateRoom,
			Data: mustMarshal(room),
		}
		cmdBytes := mustMarshal(cmd)

		go func(cmdBytes []byte, room shared.GameRoom) {
			future := fsm.Raft.Apply(cmdBytes, 5*time.Second)
			if err := future.Error(); err != nil {
				log.Printf("[FSM] Erro ao aplicar CREATE_ROOM via Raft: %v", err)
				return
			}
			log.Printf("[FSM] Sala global criada: %s (%s vs %s) - Host: server%d",
				room.ID, room.Player1.UserName, room.Player2.UserName, room.ServerID)

		}(cmdBytes, room)

		log.Printf("[FSM] Preparando sala global: %s (%s vs %s) - Host: server%d",
			room.ID, player1.UserName, player2.UserName, room.ServerID)
	}
	fsm.GlobalQueueMu.Unlock()
}

func chooseRandomPlayer(a, b string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	if n.Int64() == 0 {
		return a
	}
	return b
}

func chooseRandomPlayerInt(a, b int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	if n.Int64() == 0 {
		return a
	}
	return b
}

func (fsm *FSM) SendPlayerToLeaderQueue(entry shared.QueueEntry) {
	if fsm.Raft == nil {
		log.Println("[FSM] Raft não inicializado")
		return
	}

	// Aplica o comando no Raft
	cmd := sharedRaft.Command{
		Type: sharedRaft.CommandQueueJoinGlobal,
		Data: mustMarshal(entry),
	}

	future := fsm.Raft.Apply(mustMarshal(cmd), 5*time.Second)
	if err := future.Error(); err != nil {
		log.Printf("[FSM] Erro ao enviar jogador para líder: %v", err)
		return
	}

	log.Printf("[FSM] Jogador %s enviado para líder", entry.Player.UserName)
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}


func (fsm *FSM) ApplyPlayCard(room *shared.GameRoom, playerID string, card shared.Card) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	// Atualiza turno, cartas jogadas, etc.
	if room.Turn != playerID {
		log.Printf("Não é a vez do jogador %s", playerID)
		return
	}
	log.Println("Jogou essa porra de carta: ", card)
}



