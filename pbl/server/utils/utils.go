package utils

import (
	"fmt"
	"net"
	"bytes"
	"time"
    "encoding/json"
    "net/http"
	"pbl/server/models"
)

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

//PODE APAGAR ISSO AQUI --> TAVA SENDO USADO NO BULLY
//Para enciar a mensagem de eleição 
func SendElectionMessage(peerURL string, message models.ElectionMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("erro ao codificar: %w", err)
	}


	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(peerURL+"/election", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("erro ao enviar POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status não OK: %d", resp.StatusCode)
	}

	return nil
}