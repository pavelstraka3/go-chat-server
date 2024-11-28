package main

import (
	"fmt"
	"net/http"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	db := connectDB()
	defer db.Close()

	mux := http.NewServeMux()
	createUserTable(db)

	manager := NewClientManager()

	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/ws", handleWebSocket(manager))
	mux.HandleFunc("POST /register", handleRegisterUser(db))
	mux.HandleFunc("POST /login", handleLoginUser(db))
	mux.HandleFunc("GET /users", handleGetUsers(db))

	fmt.Println("Server started on port 8090")
	if err := http.ListenAndServe(":8090", corsMiddleware(mux)); err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
