package utils

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"pbl/shared"
	"pbl/style"
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

func ShowWaitingScreen(user shared.User, matchChan <-chan bool, doneChan <-chan struct{}) {
    style.Clear()
	fmt.Printf("Olá %s! Entrou na fila. Aguardando pareamento...\n", user.UserName)
    
    spinner := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	i := 0

    for {
        select {
        case <-doneChan:
            //fmt.Println("\nParabéns! Match encontrado! Preparando partida...")
            return
        default:
            fmt.Printf("\r%s Aguardando pareamento...", spinner[i%len(spinner)])
            i++
            time.Sleep(300 * time.Millisecond)
        }
    }
}
