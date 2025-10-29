package game

import "pbl/shared"

type MatchInfo struct {
    Opponent shared.User
    Room     shared.GameRoom
    IsGlobal bool
}
