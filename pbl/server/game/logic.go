package game

import (
	"log"
	"pbl/shared"
)

func CheckWinner(card1, card2 shared.Card) string{
	//O retorno é o resultado do player 1, depois é apenas comparado: se player 1 perdeu, o player 2 ganhou. Caso o contrário, o player 2 perdeu. E em caso de empate, vai ser empate para os dois. 
	log.Printf("\033[31mCarta P1: %s\033[0m", card1.Element)
	log.Printf("\033[31mCarta P2: %s\033[0m", card2.Element)
	
	switch card1.Element{
	case "FOGO":
		switch card2.Element{
		case "AR", "FOGO":
			return "EMPATE"
		case "TERRA":
			return "GANHOU"
		case "AGUA", "MATO":
			return "PERDEU"
		}
	case "AGUA":
		switch card2.Element {
		case "TERRA", "AGUA":
			return "EMPATE"
		case "FOGO":
			return "GANHOU"
		case "AR", "MATO":
			return "PERDEU"
		}
	case "TERRA":
		switch card2.Element {
		case "AGUA", "TERRA":
			return "EMPATE"
		case "AR":
			return "GANHOU"
		case "FOGO", "MATO":
			return "PERDEU"
		}
	case "AR":
		switch card2.Element {
		case "FOGO", "AR":
			return "EMPATE"
		case "AGUA":
			return "GANHOU"
		case "TERRA", "MATO":
			return "PERDEU"
		}
	case "MATO": //carta do mal. Só empata ou perde✨
	//Percebi agora que é só o usuário não colocar essa carta no deck...
		switch card2.Element {
		case "MATO":
			return "EMPATE"
		case "TERRA", "AGUA", "FOGO", "AR":
			return "PERDEU"
		}
	}
	return ""
}

