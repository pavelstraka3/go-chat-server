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
	creatRoomTable(db)
	createMessageTable(db)

	manager := NewClientManager(db)

	mux.HandleFunc("/api/ping", ping)
	mux.HandleFunc("/api/ws", handleWebSocket(manager))
	mux.HandleFunc("POST /api/register", handleRegisterUser(db))
	mux.HandleFunc("POST /api/login", handleLoginUser(db))
	mux.HandleFunc("GET /api/users", handleGetUsers(db))
	mux.HandleFunc("GET /api/messages", handleGetMessages(db))

	// serving react frontend
	//fs := http.FileServer(http.Dir("build"))

	fmt.Println("Server started on port 8090")
	if err := http.ListenAndServe(":8090", corsMiddleware(mux)); err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
