package cards

import (
	"math/rand"
	"pbl/shared"

	"github.com/google/uuid"
)

//var elements = []string{"AGUA", "TERRA", "FOGO", "AR", "MATO"}

var typesElemnts = map[string][]string{
	"AGUA": {"BENTA", "LIMPA", "GLACIAL", "DOCE", "FORTE", "SALGADA", "MOLHADA"},
	"FOGO": {"SAGRADO", "HADES", "ETERNO", "DRAGÃO", "FORTE", "AZUL", "FRIO"},
	"TERRA": {"DIAMENTE", "VULCÂNICA", "FERTIL", "VERMELHA", "FORTE", "BATIDA", "SOLTA"},
	"AR": {"FEDENDO", "POLUIDO", "LIMPO", "TORNADO", "FORTE", "CHEIROSO", "CONDICIONADO"},
}

func GerarEstoque() []shared.Card {
	var estoque []shared.Card
	for elemento, tipos := range typesElemnts {

		for range 8{
			novaCartaNormal := shared.Card{
				Element: elemento,
				Type: "NORMAL",
				Id: uuid.New().String(),
			}
			estoque = append(estoque, novaCartaNormal)
			novaCartaRara := shared.Card{
				Element: elemento,
				Type: tipos[rand.Intn(len(tipos))],
				Id: uuid.New().String(),
			}
			estoque = append(estoque, novaCartaRara)
		}
	}
	rand.Shuffle(len(estoque), func(i, j int) {
		estoque[i], estoque[j] = estoque[j], estoque[i]
	})
	
	return estoque
}
