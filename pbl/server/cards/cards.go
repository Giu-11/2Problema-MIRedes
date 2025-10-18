package cards

import (
	"pbl/shared"

	"github.com/google/uuid"
)

func GerarEstoque() []shared.Card {
	var estoque []shared.Card
	for range 10 {
		newCard := shared.Card{Element: "ar", Type: "normal"}
		newCard.Id = uuid.New().String()
		estoque = append(estoque, newCard)
	}
	return estoque
}
