// pbl/shared/raft.go
package sharedRaft

import "encoding/json"

// tipos de comando que podemos usar no log do Raft
// const CommandRegisterUser = "REGISTER_USER"
const CommandLoginUser = "LOGIN"

// command representa uma ação a ser aplicada na maquina de estados
type Command struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
