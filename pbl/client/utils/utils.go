package utils

import(
	"os"
    "fmt"
	"math"
	"time"
	"bufio"
    "strings"
	"os/exec"
	"runtime"
	"crypto/rand"
	"encoding/binary"

	"pbl/shared"
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

func GerarIdAleatorio() int {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := int(binary.LittleEndian.Uint32(b[:]))
	fmt.Println("ID:", id)
	return int(math.Abs(float64(id)))
}

func ShowWaitingScreen(user shared.User, matchChan chan bool) {
    fmt.Printf("Olá %s! Entrou na fila. Aguardando pareamento...\n", user.UserName)
    
    spinner := []string{"|", "/", "-", "\\"}
    i := 0

    for {
        select {
        case <-matchChan:
            fmt.Println("\nParabéns! Match encontrado! Preparando partida...")
            return
        default:
            fmt.Printf("\r%s Aguardando pareamento...", spinner[i%len(spinner)])
            i++
            time.Sleep(300 * time.Millisecond)
        }
    }
}