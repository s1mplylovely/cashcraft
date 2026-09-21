package main

import (
	"fmt"
	"log"
	"net/http"
)

func pingHandler(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(w, "Недопустимый метод", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "pong")
}

func main() {
	http.HandleFunc("/ping", pingHandler)

	port := ":8080"
	log.Printf("Gateway запущен на %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Неудалось запустить сервер: %v", err)
	}
}
