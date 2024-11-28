package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity (not recommended in production).
	},
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello!")
}

func handleWebSocket(manager *ClientManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from query parameters
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		// Validate the JWT token
		username, err := validateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Upgrade the HTTP connection to a WebSocket connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading to WebSocket: ", err)
			return
		}
		defer conn.Close()

		clientID := uuid.New().String()
		client := &Client{
			Conn:     conn,
			Username: username,
		}
		manager.AddClient(clientID, client)

		manager.JoinRoom("general", client)
		sendMessage(conn, SystemMessage, "You have joined the room: general", "system", "general")

		// Notify room members
		manager.BroadcastMessageToRoom("general", []byte(fmt.Sprintf("%s has joined the room.", username)), "system")

		// Listen for messages from client
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from %s\n %s: ", username, err)
				break
			}

			// parse the message
			parsedMessage := parseMessage(string(message), client.ChatRoom.Name)

			// Check if the client has joined a room
			if client.ChatRoom == nil && parsedMessage.Type != CommandMessage {
				sendMessage(conn, SystemMessage, "You must join a room first. Use /join <roomName>", "system", "")
				continue
			}

			switch parsedMessage.Type {
			case RegularMessage:
				log.Printf("[%s]: %s\n", username, parsedMessage.Content)
				manager.BroadcastMessageToRoom(client.ChatRoom.Name, []byte(fmt.Sprintf(parsedMessage.Content)), username)

			case DirectMessage:
				log.Printf("[DM from %s to %s]: %s\n", username, parsedMessage.Target, parsedMessage.Content)
				targetClient := manager.FindClientByUsername(parsedMessage.Target)
				if targetClient != nil {
					sendMessage(targetClient.Conn, DirectMessage, parsedMessage.Content, username, "")
				} else {
					sendMessage(conn, SystemMessage, fmt.Sprintf("User %s not found.", parsedMessage.Target), "system", "")
				}
			case CommandMessage:
				switch parsedMessage.Command {
				case UsersCommand:
					var sb strings.Builder
					for key, val := range manager.Usernames {
						if val {
							sb.WriteString(key + "\n")
						}
					}
					sendMessage(conn, SystemMessage, sb.String(), "system", "")
				case JoinCommand:
					roomName := parsedMessage.Content
					manager.JoinRoom(roomName, client)
					sendMessage(conn, SystemMessage, fmt.Sprintf("You have joined the room: %s", roomName), "system", roomName)

					for _, msg := range client.ChatRoom.History {
						if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
							fmt.Println("Error sending history: ", err)
							return
						}
					}
					// Notify room members
					manager.BroadcastMessageToRoom(roomName, []byte(fmt.Sprintf("%s has joined the room.", username)), "system")
				default:
					sendMessage(conn, SystemMessage, "Invalid command. Use /help for a list of commands.", "system", "")
				}
			case InvalidMessage:
				sendMessage(conn, SystemMessage, parsedMessage.Content, "system", "")
			}
		}

		// Cleanup when the client disconnects
		manager.RemoveClient(clientID)
		manager.ReleaseUsername(username)
		manager.BroadcastMessage([]byte(fmt.Sprintf("[Server]: %s has left the chat.", username)))
	}
}

func handleRegisterUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// register the user
		err = registerUser(db, user.Username, user.Password)
		if err != nil {
			http.Error(w, "Registration failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("User registered successfully"))
	}
}

func handleLoginUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		isValid, err := loginUser(db, user.Username, user.Password)
		if err != nil {
			http.Error(w, "Login failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !isValid {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Gnerate JWT token
		token, err := generateJWT(user.Username)
		if err != nil {
			http.Error(w, "Failed to generate JWT token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})
	}
}

func handleGetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := getAllUsers(db)
		if err != nil {
			http.Error(w, "Failed to get users: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}
