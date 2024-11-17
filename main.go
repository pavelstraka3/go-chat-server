package main

import (
	"fmt"
	"net/http"
)

func main() {
	manager := NewClientManager()

	http.HandleFunc("/ping", ping)
	http.HandleFunc("/ws", handleWebSocket(manager))

	fmt.Println("Server started on port 8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
