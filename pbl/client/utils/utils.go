package utils

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strings"
	//"time"
	//"pbl/shared"
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

func Contains(slice []int, value int) bool {
    for _, v := range slice {
        if v == value {
            return true
        }
    }
    return false
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

func GenerateRoomID(serverID int) string {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := int(binary.LittleEndian.Uint32(b[:]))
	return fmt.Sprintf("%d-%d", serverID, int(math.Abs(float64(id))))
}

/*
func ShowWaitingScreen(user shared.User, matchChan <-chan shared.User) shared.User {
	fmt.Printf("\nOlá %s! Entrou na fila. Aguardando pareamento...\n", user.UserName)

	spinner := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	i := 0
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case opponent := <-matchChan:
			fmt.Printf("\nMatch encontrado! Jogando contra: %s\n", opponent.UserName)
			return opponent
		case <-ticker.C:
			fmt.Printf("\r%s Aguardando pareamento...", spinner[i%len(spinner)])
			i++
		}
	}
}
*/