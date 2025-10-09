package utils

import(
	"bufio"
    "fmt"
    "os"
    "strings"
	"os/exec"
	"runtime"
)

func ReadLineSafe() string {
    reader := bufio.NewReader(os.Stdin)
    input, err := reader.ReadString('\n')
    if err != nil {
        fmt.Println("Erro ao ler input:", err)
        return ""
    }
    return strings.TrimSpace(input)
}

func Clear() {
	nameOS := runtime.GOOS
	fmt.Println("Sistema operacional:", nameOS)

	var cmd *exec.Cmd
	if nameOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}