package utils

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"math"
	"bytes"
    "encoding/json"
    "log"
    "net/http"
	"pbl/server/models"
)

//Gerar um ID aleatório 
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

//Descobrir o IP do pc que tá rodando o servidor
func LocalIP() (string, error) {
ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("erro ao listar interfaces de rede: %v", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue
			}

			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("não foi possível detectar um IP local válido")
}

//Para enciar a mensagem de eleição 
func SendElectionMessage(peerURL string, message models.ElectionMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Erro ao codificar mensagem de eleição: %v", err)
		return
	}

	resp, err := http.Post(peerURL+"/election", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Erro ao enviar mensagem de eleição para %s: %v", peerURL, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Mensagem de eleição enviada para %s", peerURL)
}
