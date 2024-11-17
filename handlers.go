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
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Please enter your username:")); err != nil {
			log.Fatalln("Error sending message: ", err)
			return
		}

		var username string
		for {
			_, usernameMessage, err := conn.ReadMessage()
			if err != nil {
				log.Fatalln("Error reading username: ", err)
				return
			}

			username = string(usernameMessage)

			// validate the username
			isValid, reason := manager.IsValidUsername(username)

			if !isValid {
				errMsg := fmt.Sprintf("Invalid username: %s", reason)
				conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
				continue
			}

			// reserve username and proceed
			manager.ReserveUsername(username)
			log.Printf("Client connected with username: %s\n", username)

			clientID := uuid.New().String()
			client := &Client{
				Conn:     conn,
				Username: username,
			}
			manager.AddClient(clientID, client)

			// Initial room setup: Ask the client which room to join
			conn.WriteMessage(websocket.TextMessage, []byte("Please type a room name to join:"))

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
					conn.WriteMessage(websocket.TextMessage, []byte("You must join a room first. Use /join <roomName>"))
					continue
				}

				switch parsedMessage.Type {
				case RegularMessage:
					log.Printf("[%s]: %s\n", username, parsedMessage.Content)
					manager.BroadcastMessageToRoom(client.ChatRoom.Name, []byte(fmt.Sprintf("[%s]: %s", username, parsedMessage.Content)))

				case DirectMessage:
					log.Printf("[DM from %s to %s]: %s\n", username, parsedMessage.Target, parsedMessage.Content)
					targetClient := manager.FindClientByUsername(parsedMessage.Target)
					if targetClient != nil {
						targetClient.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("[DM from %s]: %s", username, parsedMessage.Content)))
					} else {
						conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("User %s not found.", parsedMessage.Target)))
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
						conn.WriteMessage(websocket.TextMessage, []byte(sb.String()))
					case JoinCommand:
						roomName := parsedMessage.Content
						manager.JoinRoom(roomName, client)
						conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("You have joined the room: %s", roomName)))

						for _, msg := range client.ChatRoom.History {
							if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
								fmt.Println("Error sending history: ", err)
								return
							}
						}
						// Notify room members
						manager.BroadcastMessageToRoom(roomName, []byte(fmt.Sprintf("[%s]: %s has joined the room.", roomName, username)))
					default:
						conn.WriteMessage(websocket.TextMessage, []byte(parsedMessage.Content))
					}
				case InvalidMessage:
					conn.WriteMessage(websocket.TextMessage, []byte(parsedMessage.Content))
				}
			}

			// Cleanup when the client disconnects
			manager.RemoveClient(clientID)
			manager.ReleaseUsername(username)
			manager.BroadcastMessage([]byte(fmt.Sprintf("[Server]: %s has left the chat.", username)))
			break
		}
	}
}
