package shared

import (
	"encoding/json"
	"time"
)

type GameMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	From string             `json:"from"`
	ServerID int             `json:"server_id"`
}

type User struct {
	UserName string   `json:"username"`
	UserId string     `json:"user_id"`
	Password string   `json:"password"`
	Cards    []Card `json:"cards"`
	Deck     []Card `json:"deck"`
	Status string
}

type Card struct {
	Element string `json:"element"`
	Type    string `json:"type"`
	Id      int    `json:"id"`
}

type Request struct {
	ClientID string          `json:"client_id"`
	Action   string          `json:"action"`
	Payload  json.RawMessage `json:"payload"`
}

type Response struct {
	Status string          `json:"status"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Server int             `json:"server"`
}
type QueueEntry struct {
	Player User    `json:"user"`
	ServerID string    `json:"server_id"`
	Topic    string    `json:"topic"`
	JoinTime time.Time `json:"join_time"`
}

//Estado da partida
type GameStatus string

const (
	WaitingPlayers GameStatus = "WAITING"
	InProgress     GameStatus = "IN_PROGRESS"
	Finished       GameStatus = "FINISHED"
)

type GameRoom struct {
	ID        string         `json:"id"`
	Player1   *User    `json:"player1"`
	Player2   *User    `json:"player2"`
	Player1ClientID string
    Player2ClientID string
	Turn      *User    `json:"turn"`
	Status    GameStatus     `json:"status"`
	Winner    *User   `json:"winner,omitempty"`
}
