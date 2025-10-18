package sharedRaft

import "encoding/json"

// tipos de comando que podemos usar no log do Raft
// const CommandRegisterUser = "REGISTER_USER"
const CommandLoginUser = "LOGIN"
const CommandOpenPack = "ABRIR_PACOTE"
const CommandClaimCard = "RESGATAR_CARTA"

// command representa uma ação a ser aplicada na maquina de estados
type Command struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// informações sobre um pedido de reservar carta
type DrawCardPayload struct {
	PlayerID  string `json:"player_id"`
	RequestID string `json:"request_id"` // identifica o pedido e garante idempotencia
}

// informações sobre um pedido de pegar a carta
type ClaimCardPayload struct {
	RequestID string `json:"request_id"`
}
