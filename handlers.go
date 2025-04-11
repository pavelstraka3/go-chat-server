package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity (not recommended in production).
	},
}

func ping(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "Hello!")
	if err != nil {
		return
	}
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
		email, err := validateJWT(token)
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
			Conn:  conn,
			Email: email,
		}
		manager.AddClient(clientID, client)

		if err := manager.JoinRoom("general", client); err != nil {
			log.Printf("Error joining room 'general': %v", err)
			if err := sendMessage(conn, SystemMessage, "Failed to join room: "+err.Error(), "system", ""); err != nil {
				log.Printf("Error sending failure message: %v", err)
			}
			return
		}

		sendMessage(conn, SystemMessage, "You have joined the room: general", "system", "general")

		// Listen for messages from client
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from %s\n %s: ", email, err)
				break
			}

			//sendMessage(conn, SystemMessage, "Recieved message: "+string(message), "system", "system")

			if err := handleClientMessage(conn, client, manager, message, email); err != nil {
				log.Printf("Error handling message: %v", err)
				sendMessage(conn, SystemMessage, "Error handling message: "+err.Error(), "system", "error")
			}
		}

		// Cleanup when the client disconnects
		manager.RemoveClient(clientID)
		manager.BroadcastMessage([]byte(fmt.Sprintf("[Server]: %s has left the chat.", email)))
	}
}

func handleClientMessage(conn *websocket.Conn, client *Client, manager *ClientManager, message []byte, email string) error {
	// parse the message
	parsedMessage := parseMessage(string(message))

	// Check if the client has joined a room
	if client.Room == nil && parsedMessage.Type != CommandMessage {
		sendMessage(conn, SystemMessage, "You must join a room first. Use /join <roomName>", "system", "")
		return nil
	}

	switch parsedMessage.Type {
	case RegularMessage:
		log.Printf("[%s]: %s\n", email, parsedMessage.Content)
		manager.BroadcastMessageToRoom(client.Room.Name, []byte(fmt.Sprintf(parsedMessage.Content)), email)
		err := saveMessageToDb(manager.Db, parsedMessage, parsedMessage.Room.ID, client.Email)
		if err != nil {
			log.Printf("Error saving message to DB: %v", err)
			sendMessage(conn, SystemMessage, "Error saving message to DB: "+err.Error(), "system", "")
		}
	case DirectMessage:
		log.Printf("[DM from %s to %s]: %s\n", email, parsedMessage.Target, parsedMessage.Content)
		targetClient := manager.FindClientByEmail(parsedMessage.Target)
		if targetClient != nil {
			sendMessage(targetClient.Conn, DirectMessage, parsedMessage.Content, email, "")
		} else {
			sendMessage(conn, SystemMessage, fmt.Sprintf("User %s not found.", parsedMessage.Target), "system", "")
		}
	case CommandMessage:
		switch parsedMessage.Command {
		case UsersCommand:
			var sb strings.Builder
			for key, val := range manager.Emails {
				if val {
					sb.WriteString(key + "\n")
				}
			}
			sendMessage(conn, SystemMessage, sb.String(), "system", "")
		case JoinCommand:
			roomName := parsedMessage.Content
			if err := manager.JoinRoom(roomName, client); err != nil {
				log.Printf("Failed to join room: %v", err)
				sendMessage(conn, SystemMessage, "Failed to join room: "+err.Error(), "system", "")
				return err
			}

			for _, msg := range client.Room.History {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					return fmt.Errorf("error sending history: %w", err)
				}
			}
			// Notify room members
			manager.BroadcastMessageToRoom(roomName, []byte(fmt.Sprintf("%s has joined the room.", email)), "system")
		default:
			sendMessage(conn, SystemMessage, "Invalid command. Use /help for a list of commands.", "system", "")
		}
	case InvalidMessage:
		sendMessage(conn, SystemMessage, parsedMessage.Content, "system", "")
	}
	return nil
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
		err = registerUser(db, user.Email, user.Password)
		if err != nil {
			http.Error(w, "Registration failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
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

		isValid, err := loginUser(db, user.Email, user.Password)
		if err != nil {
			http.Error(w, "Login failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !isValid {
			http.Error(w, "Invalid e-mail or password", http.StatusUnauthorized)
			return
		}

		// Gnerate JWT token
		token, err := generateJWT(user.Email)
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

func handleGetMessages(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomParam := r.URL.Query().Get("roomId")

		if roomParam == "" {
			http.Error(w, "Missing roomId", http.StatusBadRequest)
			return
		}

		roomId, err := strconv.Atoi(roomParam)
		if err != nil {
			http.Error(w, "Invalid roomId", http.StatusBadRequest)
			return
		}
		userEmail := r.URL.Query().Get("sender")

		messages, err := getMessages(db, roomId, userEmail)
		if err != nil {
			http.Error(w, "Failed to get messages: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}
