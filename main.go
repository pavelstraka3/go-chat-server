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
		// Get the path
		urlPath := r.URL.Path

		// Debug logging
		fmt.Printf("Received request for: %s\n", urlPath)

		// Check if the request is for an API endpoint
		if strings.HasPrefix(urlPath, "/api") {
			http.NotFound(w, r)
			return
		}

		// Set proper MIME types based on file extension
		ext := filepath.Ext(urlPath)
		switch strings.ToLower(ext) {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".mjs":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		}

		// Convert to OS-specific path format
		relFilePath := filepath.FromSlash(strings.TrimPrefix(urlPath, "/"))
		absFilePath := filepath.Join(staticPath, relFilePath)

		// Debug logging
		fmt.Printf("Looking for file at: %s\n", absFilePath)

		// Check if the file exists
		fileInfo, err := os.Stat(absFilePath)

		// If the file exists and is not a directory, serve it
		if err == nil && !fileInfo.IsDir() {
			fmt.Printf("Found and serving file: %s\n", absFilePath)
			fileServer.ServeHTTP(w, r)
			return
		}

		// If the request is for a static asset but file doesn't exist,
		// check in the assets directory
		if strings.HasPrefix(urlPath, "/assets/") {
			assetPath := filepath.Join(staticPath, "assets", filepath.Base(urlPath))
			if _, err := os.Stat(assetPath); err == nil {
				fmt.Printf("Found and serving asset: %s\n", assetPath)
				http.ServeFile(w, r, assetPath)
				return
			}
		}

		// For all other routes, serve index.html
		indexFile := filepath.Join(staticPath, indexPath)
		fmt.Printf("Serving index.html for path: %s\n", urlPath)
		http.ServeFile(w, r, indexFile)
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
	mux.HandleFunc("GET /api/rooms", handleGetRooms(db))

	// Verify static directory exists
	buildDir := "./static"
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
