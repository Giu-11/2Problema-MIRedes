package shared

import "encoding/json"

type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	From int             `json:"from"`
}

type User struct {
	UserName string   `json:"username"`
	Password string   `json:"password"`
	Cards    []string `json:"cards"`
	Deck     []string `json:"deck"`
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
