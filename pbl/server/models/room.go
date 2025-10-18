package models

import (
	"pbl/shared"
)

//Estado da partida
type GameStatus string

const (
	WaitingPlayers GameStatus = "WAITING"
	InProgress     GameStatus = "IN_PROGRESS"
	Finished       GameStatus = "FINISHED"
)

type Room struct {
	ID        string         `json:"id"`
	Player1   shared.User    `json:"player1"`
	Player2   shared.User    `json:"player2"`
	Turn      shared.User    `json:"turn"`
	Status    GameStatus     `json:"status"`
	Winner    *shared.User   `json:"winner,omitempty"`
}
