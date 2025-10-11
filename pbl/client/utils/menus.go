package utils

import (
	"fmt"
)

//MENUS PARA BASE - sujeito a mudanças

// Pra poder usar cores no terminal
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
)

func EscolherServidor() string {
	fmt.Println("------------------------------")
	fmt.Println("       Escolher servidor      ")
	fmt.Println("------------------------------")
	fmt.Println("1 - Servidor 1") //seria legal nomear os servidores
	fmt.Println("2 - Servidor 2")
	fmt.Println("3 - Servidor 2")
	fmt.Println("Insira o servidor que deseja: ")
	input := ReadLineSafe()
	return input
}

func MenuInicial() string {
	fmt.Println("\n----------------------------------")
	fmt.Println("           MENU INICIAL           ")
	fmt.Println("----------------------------------")
	fmt.Println("1 - Cadastro")
	fmt.Println("2 - Login")
	fmt.Println("3 - Sair")
	fmt.Print("Insira a opção desejada: ")
	return ReadLineSafe()

}

func ShowMenuLogin() {
	for {
		fmt.Println("\n----------------------------------")
		fmt.Println("               Menu               ")
		fmt.Println("----------------------------------")
		fmt.Println("1 - Entrar na fila")
		fmt.Println("2 - Ver/alterar deck")
		fmt.Println("3 - Abrir pacote")
		fmt.Println("4 - Visualizar regras")
		fmt.Println("5 - Visualizar ping")
		fmt.Println("6 - Deslogar")
		fmt.Print("Insira a opção desejada: ")
		//option := ReadLineSafe()
		//return input
	}
}

func ShowRules() {
	fmt.Println("\n----------------------------------")
	fmt.Println("              Regras              ")
	fmt.Println("----------------------------------")
	fmt.Println("")
	//tem que escolher o jogo

}
