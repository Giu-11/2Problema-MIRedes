package utils

import (
	"fmt"
	"pbl/shared"
	"pbl/style"
)

//MENUS PARA BASE - sujeito a mudanças

func EscolherServidor() string {
	fmt.Println("------------------------------")
	fmt.Println("       Escolher servidor      ")
	fmt.Println("------------------------------")
	fmt.Println("1 - Servidor 1") //seria legal nomear os servidores
	fmt.Println("2 - Servidor 2")
	fmt.Println("3 - Servidor 3")
	fmt.Println("Insira o servidor que deseja: ")
	input := ReadLineSafe()
	return input
}

func MenuInicial() string {
	fmt.Println("\n----------------------------------")
	fmt.Println("           MENU INICIAL           ")
	fmt.Println("----------------------------------")
	fmt.Println("1 - Login")
	fmt.Println("2 - Sair")
	fmt.Print("Insira a opção desejada: ")
	return ReadLineSafe()

}

func ShowMenuPrincipal() string {
	fmt.Println("\n----------------------------------")
	fmt.Println("               Menu               ")
	fmt.Println("----------------------------------")
	fmt.Println("1 - Entrar na fila")
	fmt.Println("2 - Ver/alterar deck")
	fmt.Println("3 - Abrir pacote")
	fmt.Println("4 - Trocar cartas")
	fmt.Println("5 - Visualizar regras")
	fmt.Println("6 - Visualizar ping") //precisa?
	fmt.Println("7 - Deslogar")
	fmt.Print("Insira a opção desejada: ")
	return ReadLineSafe()
}

func ShowRules() {
	fmt.Println("\n----------------------------------")
	fmt.Println("              Regras              ")
	fmt.Println("----------------------------------")
	fmt.Println("Ao fazer o cadastro você recebeu\n5 cartas. Sendo elas: AGUA, TERRA,\nFOGO, AR e MATO")
	fmt.Println("\nCada carta tem seus pontos fortes\ne fracos:")
	fmt.Println("\n ÁGUA")
	fmt.Println(" Forte contra FOGO")
	fmt.Println(" Fraco contra AR")

	fmt.Println("\n TERRA")
	fmt.Println(" Forte contra AR")
	fmt.Println(" Fraco contra FOGO")

	fmt.Println("\n FOGO")
	fmt.Println(" Forte contra TERRA")
	fmt.Println(" Fraco contra ÁGUA")

	fmt.Println("\n AR")
	fmt.Println(" Forte contra ÁGUA")
	fmt.Println(" Fraco contra TERRA")

	fmt.Println("\n MATO")
	fmt.Println(" Carta MISTERIOSA")

	fmt.Println("----------------------------------")

}

func ShowMenuCards()string{
	fmt.Println("\n----------------------------------")
	fmt.Println("            Menu Cartas             ")
	fmt.Println("----------------------------------")
	fmt.Println("1 - Ver cartas")
	fmt.Println("2 - Mudar deck")
	fmt.Println("3 - Ver deck atual")
	fmt.Println("4 - Voltar ao menu principal")
	return ReadLineSafe()
}


func PrintCartaCor(carta shared.Card){
	cardString := fmt.Sprintf("%s %s", carta.Element, carta.Type)
	switch carta.Element {
		case "AR":
			style.PrintCian(cardString)
		case "AGUA":
			style.PrintAz(cardString)
		case "FOGO":
			style.PrintVerm(cardString)
		case "TERRA":
			style.PrintAma(cardString)
		case "MATO":
			style.PrintVerd(cardString)
		}
}


func MostrarInventario(cartas []shared.Card){
	fmt.Println("\n----------------------------------")
	fmt.Println("             Suas Cartas             ")
	fmt.Println("----------------------------------")
	for i,carta := range cartas{
		fmt.Printf("[%d] - ", i)
		PrintCartaCor(carta)
		fmt.Print("\n")
	}
}

//Printar as cartas do deck do usuário
func ListCardsDeck(user *shared.User) {
	fmt.Println("\n----------------------------------")
	fmt.Println("             Seu deck             ")
	fmt.Println("----------------------------------")
	for i,carta := range user.Deck{
		fmt.Printf("[%d] - ", i)
		PrintCartaCor(carta)
		fmt.Print("\n")
	}
	fmt.Println("----------------------------------")
}