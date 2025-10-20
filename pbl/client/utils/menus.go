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
	fmt.Println("")
	//tem que escolher o jogo

}

func MostrarInventario(cartas []shared.Card){
	fmt.Println("Suas cartas:")
	for i,carta := range cartas{
		msg := fmt.Sprintf("%d - %s %s\n", i, carta.Element, carta.Type)
		switch carta.Element {
		case "AR":
			style.PrintCian(msg)
		case "AGUA":
			style.PrintAz(msg)
		case "FOGO":
			style.PrintVerm(msg)
		case "TERRA":
			style.PrintAma(msg)
		case "MATO":
			style.PrintVerd(msg)
		}
	}
}
