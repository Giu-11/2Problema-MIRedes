package game

import "strings"

//peguei do problema anterior --> tem que adaptar
func CheckWinner(card1, card2 string) string{
	//O retorno é o resultado do player 1, depois é apenas comparado: se player 1 perdeu, o player 2 ganhou. Caso o contrário, o player 2 perdeu. E em caso de empate, vai ser empate para os dois. 

	switch card1{
	case "FOGO":
		switch card2 {
		case "AR", "FOGO":
			return "EMPATE"
		case "TERRA":
			return "GANHOU"
		case "AGUA", "MATO":
			return "PERDEU"
		}
	case "AGUA":
		switch card2 {
		case "TERRA", "AGUA":
			return "EMPATE"
		case "FOGO":
			return "GANHOU"
		case "AR", "MATO":
			return "PERDEU"
		}
	case "TERRA":
		switch card2 {
		case "AGUA", "TERRA":
			return "EMPATE"
		case "AR":
			return "GANHOU"
		case "FOGO", "MATO":
			return "PERDEU"
		}
	case "AR":
		switch card2 {
		case "FOGO", "AR":
			return "EMPATE"
		case "AGUA":
			return "GANHOU"
		case "TERRA", "MATO":
			return "PERDEU"
		}
	case "MATO": //carta do mal. Só empata ou perde✨ //Essa carta vai estar nos pacotes.
	//Percebi agora que é só o usuário não colocar essa carta no deck...
		switch card2 {
		case "MATO":
			return "EMPATE"
		case "TERRA", "AGUA", "FOGO", "AR":
			return "PERDEU"
		}
	}
	return ""
}

//Para não ter diferença entre cartas normais e especiais(elas são meramente ilustrativas)
func baseCard(card string) string {
    parts := strings.Fields(card) 
    if len(parts) > 0 {
        return parts[0]
    }
    return card
}