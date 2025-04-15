package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func SpaHandler(staticPath string, indexPath string) http.Handler {
	fileServer := http.FileServer(http.Dir(staticPath))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the path, but remove leading slash for Windows compatibility
		urlPath := r.URL.Path

		// Check if the request is for an API endpoint
		if strings.HasPrefix(urlPath, "/api") {
			http.NotFound(w, r)
			return
		}

		// Convert to OS-specific path format
		relFilePath := filepath.FromSlash(strings.TrimPrefix(urlPath, "/"))

		// Check if the file exists
		absFilePath := filepath.Join(staticPath, relFilePath)

		fileInfo, err := os.Stat(absFilePath)

		// Debug path processing
		fmt.Printf("Request URL: %s\n", urlPath)
		fmt.Printf("Looking for file: %s\n", absFilePath)

		// If file doesn't exist or is a directory, serve index.html
		if os.IsNotExist(err) || (err == nil && fileInfo.IsDir()) {
			// Serve the index file
			indexFile := filepath.Join(staticPath, indexPath)
			http.ServeFile(w, r, indexFile)
			return
		}

		// If error other than not exists, log it
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error checking file: %v\n", err)
		}

		// Otherwise, serve the file
		fileServer.ServeHTTP(w, r)
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

	//go func() {
	//	typingTimeout := 3 * time.Second
	//	for {
	//		time.Sleep(1 * time.Second)
	//		manager.Lock.Lock()
	//		now := time.Now()
	//		for _, client := range manager.Clients {
	//			if client.IsTyping && now.Sub(client.LastTyping) > typingTimeout {
	//				// Reset typing status if timeout elapsed
	//				client.IsTyping = false
	//				manager.UpdateClientTypingStatus(client, false)
	//			}
	//		}
	//		manager.Lock.Unlock()
	//	}
	//}()

	mux.HandleFunc("/api/ping", ping)
	mux.HandleFunc("/api/ws", handleWebSocket(manager))
	mux.HandleFunc("POST /api/register", handleRegisterUser(db))
	mux.HandleFunc("POST /api/login", handleLoginUser(db))
	mux.HandleFunc("GET /api/users", handleGetUsers(db))
	mux.HandleFunc("GET /api/messages", handleGetMessages(db))
	mux.HandleFunc("GET /api/rooms", handleGetRooms(db))

	// Verify build directory exists
	buildDir := "./build"
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		fmt.Printf("Warning: Build directory '%s' doesn't exist\n", buildDir)
		os.Mkdir(buildDir, 0755) // Create it if it doesn't exist
	}

	// Add some debug info
	fmt.Printf("Working directory: %s\n", getCurrentDir())
	fmt.Printf("Build directory absolute path: %s\n", getAbsPath(buildDir))

	// Handle all non-API paths with the SPA handler
	staticFileHandler := SpaHandler(buildDir, "index.html")
	mux.Handle("/", staticFileHandler)

	fmt.Println("Server started on port 8090")
	if err := http.ListenAndServe(":8090", corsMiddleware(mux)); err != nil {
		fmt.Println("Error starting server: ", err)
	}
}

// Helper function to get current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("Error getting current directory: %v", err)
	}
	return dir
}

// Helper function to get absolute path
func getAbsPath(relPath string) string {
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return fmt.Sprintf("Error getting absolute path: %v", err)
	}
	return absPath
}
