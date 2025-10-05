package main

import (
	"fmt"
	"net/http"
)

//Endpoint simples para teste de comunicação
func PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprintln(w, "Pong from this server!")
}
