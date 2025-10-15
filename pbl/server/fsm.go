package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	sharedRaft "pbl/server/shared"
	"pbl/shared"
	"sync"

	"github.com/hashicorp/raft"
)

// maquina de estados finitos
type FSM struct {
	mu    sync.Mutex
	users map[string]shared.User // mapa de usuarios
}

func NewFSM() *FSM {
	return &FSM{
		users: make(map[string]shared.User),
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
	case sharedRaft.CommandRegisterUser:
		var user shared.User
		if err := json.Unmarshal(cmd.Data, &user); err != nil {
			log.Printf("[FSM] ERROR: failed to unmarshal user data: %s", err)
			return err
		}
		log.Printf("[FSM] Applying command: Register User '%s'", user.UserName)
		fsm.users[user.UserName] = user
		return nil
	default:
		return fmt.Errorf("unrecognized command type: %s", cmd.Type)
	}
}

// cria saves do estado do servidor
// por enquanto somente os usuarios
func (fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	usersCopy := make(map[string]shared.User)
	for k, v := range fsm.users {
		usersCopy[k] = v
	}
	return &fsmSnapshot{users: usersCopy}, nil
}

// Restaura a fsm a partir de um snapshot
func (fsm *FSM) Restore(rc io.ReadCloser) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	var users map[string]shared.User
	if err := json.NewDecoder(rc).Decode(&users); err != nil {
		return err
	}
	fsm.users = users
	return nil
}

// snapshot do estado do servidor
type fsmSnapshot struct {
	users map[string]shared.User
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Codifica e salva o snapshot escrevendo no disco
		if err := json.NewEncoder(sink).Encode(s.users); err != nil {
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