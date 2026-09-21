package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	username := os.Getenv("USER") // linux
	if username == "" {
		username = os.Getenv("USERNAME") // windows
	}
	fmt.Printf("Имя пользователя: %s\n", username)

	args := os.Args[1:]
	fmt.Printf("Аргументы CLI: %s\n", args)

	fmt.Printf("Текущая версия Go: %s\n", runtime.Version())
}
