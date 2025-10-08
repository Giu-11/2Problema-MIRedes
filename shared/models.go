package models

import "encoding/json"

type Event struct {
    Type string          `json:"type"` 
    Data json.RawMessage `json:"data"` 
    From int             `json:"from"`
}
