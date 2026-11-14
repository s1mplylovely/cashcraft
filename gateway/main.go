package main

import (
	"log"
	"net/http"

	"gateway/internal/api"
	"ledger/domain"
)

func main() {
	// подключаемся к той же инфраструктуре, что и Ledger
	domain.MustConnectInfra()

	port := ":8080"
	log.Printf("Gateway запущен на %s", port)
	if err := http.ListenAndServe(port, api.NewRouter()); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
