package style

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// limpa o prompt de comando
func Clear() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// mostra texto passado em vermelho
func PrintVerm(texto string) {
	fmt.Printf("\033[31m%s\033[0m", texto)
}

// mostra texto passado em verde
func PrintVerd(texto string) {
	fmt.Printf("\033[32m%s\033[0m", texto)
}

// mostra texto passado em magenta
func PrintMag(texto string) {
	fmt.Printf("\033[35m%s\033[0m", texto)
}

// mostra texto passado em ciano
func PrintCian(texto string) {
	fmt.Printf("\033[36m%s\033[0m", texto)
}

// mostra texto passado em amarelo
func PrintAma(texto string) {
	fmt.Printf("\033[33m%s\033[0m", texto)
}

func PrintAz(texto string) {
	fmt.Printf("\033[34m%s\033[0m", texto)
}
