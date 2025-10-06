package utils

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
)

func GerarIdAleatorio() int64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := int64(binary.LittleEndian.Uint64(b[:]))
	fmt.Println("ID:", id)
	return id
}

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