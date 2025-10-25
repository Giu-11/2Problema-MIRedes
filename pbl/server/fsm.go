package main

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
	globalQueue []shared.QueueEntry
	globalRooms map[string]*shared.GameRoom

	Raft *raft.Raft
}

func NewFSM() *FSM {
	return &FSM{
		users:        make(map[string]shared.User),
		cardStock:    cards.GerarEstoque(),
		pendingCards: make(map[string]shared.Card),
		globalRooms:  make(map[string]*shared.GameRoom),
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

	case sharedRaft.CommandQueueJoin:
		var entry shared.QueueEntry
		if err := json.Unmarshal(cmd.Data, &entry); err != nil {
			log.Printf("[FSM] Erro ao decodificar dados da fila global %v", err)
			return err
		}
		fsm.globalQueue = append(fsm.globalQueue, entry)
		log.Printf("[FSM] Usuário %s adicionado à fila global", entry.Player.UserName)
		//Mostra a fila global atual
		names := make([]string, len(fsm.globalQueue))
		for i, e := range fsm.globalQueue {
			names[i] = e.Player.UserName
		}
		log.Printf("[FSM] Fila global atual: %v", names)

		fsm.tryMatchPlayers()
		return nil

	case sharedRaft.CommandCreateRoom:
		var room shared.GameRoom
		if err := json.Unmarshal(cmd.Data, &room); err != nil {
			log.Printf("[FSM] Erro ao criar sala: %v", err)
			return err
		}
		fsm.globalRooms[room.ID] = &room
		log.Printf("[FSM] Sala criada: %s (%s vs %s)", room.ID, room.Player1.UserName, room.Player2.UserName)
		return nil

	case sharedRaft.CommandQueueLeave:
		var entry shared.QueueEntry
		if err := json.Unmarshal(cmd.Data, &entry); err != nil {
			log.Printf("[FSM] Erro ao decodificar LEAVE_QUEUE: %v", err)
			return err
		}
		for i, e := range fsm.globalQueue {
			if e.Player.UserId == entry.Player.UserId {
				fsm.globalQueue = append(fsm.globalQueue[:i], fsm.globalQueue[i+1:]...)
				log.Printf("[FSM] Usuário %s removido da fila global", entry.Player.UserName)
				break
			}
		}
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

func (fsm *FSM) tryMatchPlayers() {
	// vamos construir uma lista de comandos a aplicar fora do lock
	var cmds [][]byte

	for len(fsm.globalQueue) >= 2 {
		player1 := fsm.globalQueue[0].Player
		player2 := fsm.globalQueue[1].Player

		// remove da fila
		fsm.globalQueue = fsm.globalQueue[2:]

		// escolhe quem começa
		var turn string
		n, _ := rand.Int(rand.Reader, big.NewInt(2))
		if n.Int64() == 0 {
			turn = player1.UserId
		} else {
			turn = player2.UserId
		}

		room := shared.GameRoom{
			ID:      fmt.Sprintf("global-%s-vs-%s", player1.UserName, player2.UserName),
			Player1: &player1,
			Player2: &player2,
			Turn:    turn,
		}

		//cria comando CREATE_ROOM
		roomData, _ := json.Marshal(room)
		cmd := sharedRaft.Command{
			Type: sharedRaft.CommandCreateRoom,
			Data: roomData,
		}
		cmdBytes, _ := json.Marshal(cmd)

		cmds = append(cmds, cmdBytes)

		log.Printf("[FSM] Preparando criação da sala global: %s (%s vs %s)", room.ID, player1.UserName, player2.UserName)
	}
	if len(cmds) > 0 && fsm.Raft != nil {
		for _, cb := range cmds {
			cb := cb // captura
			go func() {
				future := fsm.Raft.Apply(cb, 5*time.Second)
				if err := future.Error(); err != nil {
					log.Printf("[FSM] Erro ao aplicar CREATE_ROOM via Raft: %v", err)
					return
				}
				log.Printf("[FSM] CREATE_ROOM replicado com sucesso")
			}()
		}
	}
}
