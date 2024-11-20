package main

import (
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
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading to WebSocket: ", err)
			return
		}
		defer conn.Close()

		// Request username from client
		if err := sendMessage(conn, SystemMessage, "Please enter your username:", "system"); err != nil {
			log.Println("Error sending message: ", err)
			return
		}

		// Read the username
		_, usernameMessage, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading username: %v", err)
			return
		}

		username := string(usernameMessage)

		// Validate the username
		isValid, reason := manager.IsValidUsername(username)
		if !isValid {
			errMsg := fmt.Sprintf("Invalid username: %s", reason)
			sendMessage(conn, SystemMessage, errMsg, "system")
			return
		}

		// Reserve the username
		manager.ReserveUsername(username)
		log.Printf("Client connected with username: %s\n", username)

		clientID := uuid.New().String()
		client := &Client{
			Conn:     conn,
			Username: username,
		}
		manager.AddClient(clientID, client)

		// Initial room setup: Ask the client which room to join
		if err := sendMessage(conn, SystemMessage, "Please type a room name to join:", "system"); err != nil {
			log.Printf("Error sending message: %v", err)
			return
		}

		// Listen for messages from client
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from %s\n %s: ", username, err)
				break
			}

			// parse the message
			parsedMessage := parseMessage(string(message))

			// Check if the client has joined a room
			if client.ChatRoom == nil && parsedMessage.Type != CommandMessage {
				sendMessage(conn, SystemMessage, "You must join a room first. Use /join <roomName>", "system")
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
					sendMessage(targetClient.Conn, ChatMessage, parsedMessage.Content, username)
				} else {
					sendMessage(conn, SystemMessage, fmt.Sprintf("User %s not found.", parsedMessage.Target), "system")
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
					sendMessage(conn, SystemMessage, sb.String(), "system")
				case JoinCommand:
					roomName := parsedMessage.Content
					manager.JoinRoom(roomName, client)
					sendMessage(conn, SystemMessage, fmt.Sprintf("You have joined the room: %s", roomName), "system")

					for _, msg := range client.ChatRoom.History {
						if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
							fmt.Println("Error sending history: ", err)
							return
						}
					}
					// Notify room members
					manager.BroadcastMessageToRoom(roomName, []byte(fmt.Sprintf("%s has joined the room.", username)), "system")
				default:
					sendMessage(conn, SystemMessage, "Invalid command. Use /help for a list of commands.", "system")
				}
			case InvalidMessage:
				sendMessage(conn, SystemMessage, parsedMessage.Content, "system")
			}
		}

		// Cleanup when the client disconnects
		manager.RemoveClient(clientID)
		manager.ReleaseUsername(username)
		manager.BroadcastMessage([]byte(fmt.Sprintf("[Server]: %s has left the chat.", username)))
	}
}
