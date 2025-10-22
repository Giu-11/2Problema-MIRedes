package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"pbl/server/cards"
	sharedRaft "pbl/server/shared"
	"pbl/shared"
	"sync"

	"github.com/hashicorp/raft"
)

// maquina de estados finitos
type FSM struct {
	mu           sync.Mutex
	users        map[string]shared.User // mapa de usuarios
	cardStock    []shared.Card
	pendingCards map[string]shared.Card
}

func NewFSM() *FSM {
	return &FSM{
		users:        make(map[string]shared.User),
		cardStock:    cards.GerarEstoque(),
		pendingCards: make(map[string]shared.Card),
	}
}

//aplica um comando ao raft
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
