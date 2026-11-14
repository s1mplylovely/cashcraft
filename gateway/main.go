package main

import (
	"context"
	"log"
	"net/http"

	"gateway/internal/api"
	"ledger"
)

func main() {
	ctx := context.Background()

	svc, closeFn, err := ledger.New(ctx)
	if err != nil {
		log.Fatalf("Gateway: не удалось инициализировать Ledger: %v", err)
	}
	defer closeFn()

	port := ":8080"
	log.Printf("Gateway запущен на %s", port)
	if err := http.ListenAndServe(port, api.NewRouter(svc)); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
