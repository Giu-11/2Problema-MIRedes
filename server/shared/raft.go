package sharedRaft

import "encoding/json"

// tipos de comando que podemos usar no log do Raft
const(
	CommandOpenPack   = "ABRIR_PACOTE"
	CommandClaimCard  = "RESGATAR_CARTA"
	CommandQueueJoinGlobal  = "QUEUE_JOIN_GLOBAL"
	CommandQueueLeave = "QUEUE_LEAVE"
	CommandCreateRoom = "CREATE_ROOM"
	CommandRemoveRoom   = "REMOVE_ROOM"
)

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

type GlobalQueueJoinPayload struct{
	PlayerID string `json:"player_id"`
	ServerID string `json:"server_id"`
}

type GlobalQueueLeavePayload struct{
	PlayerID string `json:"player_id"`
	ServerID string `json:"server_id"`
}

type GlobalRoomPayload struct{
	RoomID string `json:"room_id"`
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	Server1 string `json:"server1"`
	Server2 string `json:"server2"`
}
