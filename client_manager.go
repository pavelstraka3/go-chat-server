package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

type Client struct {
	Email      string
	Conn       *websocket.Conn
	Room       *Room
	IsTyping   bool
	LastTyping time.Time
}

type ClientManager struct {
	Clients map[string]*Client
	Emails  map[string]bool
	History []string
	Rooms   map[string]*Room
	Lock    sync.Mutex
	Db      *sql.DB
}

func NewClientManager(db *sql.DB) *ClientManager {
	return &ClientManager{
		Clients: make(map[string]*Client),
		Emails:  make(map[string]bool),
		History: make([]string, 0),
		Rooms:   make(map[string]*Room),
		Db:      db,
	}
}

func (cm *ClientManager) AddClient(id string, client *Client) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	cm.Clients[id] = client
	log.Printf("Client %s - %s added. Total clients: %d\n", client.Email, id, len(cm.Clients))
}

func (cm *ClientManager) RemoveClient(id string) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	delete(cm.Clients, id)
	log.Printf("Client %s removed. Total clients: %d\n", id, len(cm.Clients))
}

func (cm *ClientManager) BroadcastMessage(message []byte) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	cm.History = append(cm.History, string(message))

	for id, client := range cm.Clients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			fmt.Printf("Error writing message to client %s: %v\n", id, err)
		}
	}
}

func (cm *ClientManager) FindClientByEmail(email string) *Client {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	for _, client := range cm.Clients {
		if client.Email == email {
			return client
		}
	}
	return nil
}

func (cm *ClientManager) GetOrCreateRoom(roomName string) *Room {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	room, exists := cm.Rooms[roomName]
	if !exists {
		// if the room doesn't exist, create it
		room = &Room{
			Name:    roomName,
			Clients: make(map[string]*Client),
		}
		cm.Rooms[roomName] = room
	}

	if room.Clients == nil {
		room.Clients = make(map[string]*Client)
	}

	return room
}

func (cm *ClientManager) JoinRoom(roomName string, client *Client) (*Room, error) {
	log.Printf("Attempting to join room: %s for client: %s", roomName, client.Email)
	dbRoom, err := getRoomByName(cm.Db, roomName)
	if err != nil {
		return nil, fmt.Errorf("error getting room from database %s: %v", roomName, err)
	}

	if dbRoom == nil {
		newRoom, err := createRoom(cm.Db, roomName)
		if err != nil {
			return nil, fmt.Errorf("error creating room %s: %v", roomName, err)
		}
		dbRoom = newRoom
	}

	room, exists := cm.Rooms[roomName]
	if !exists {
		room = &Room{
			Name:    roomName,
			Clients: make(map[string]*Client),
			History: make([]string, 0),
		}
	}

	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	cm.Rooms[roomName] = room
	room.Clients[client.Email] = client
	client.Room = room

	// Notify other room members
	for email, roomClient := range room.Clients {
		if email != client.Email { // Don't notify the client who just joined
			if err := sendMessage(roomClient.Conn, SystemMessage, fmt.Sprintf("%s has joined the room.", client.Email), "system", room); err != nil {
				log.Printf("Error notifying client %s about join: %v\n", email, err)
			}
		}
	}

	log.Printf("Successfully joined room %s", roomName)
	return room, nil
}

func (cm *ClientManager) BroadcastMessageToRoom(roomName string, message []byte, user string) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	log.Printf("Broadcasting message to room %s: %s", roomName, string(message))

	room, exists := cm.Rooms[roomName]
	if !exists {
		log.Printf("Room %s does not exist", roomName)
		return
	}

	room.History = append(room.History, string(message))

	for _, client := range room.Clients {
		if err := sendMessage(client.Conn, RegularMessage, string(message), user, room); err != nil {
			log.Printf("Error sending message to client %s: %v\n", client.Email, err)
		}
	}
}

func (cm *ClientManager) UpdateClientTypingStatus(client *Client, isTyping bool) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	if client.IsTyping == isTyping {
		return
	}

	client.IsTyping = isTyping
	client.LastTyping = time.Now()

	// Don't broadcast if clien is not in room
	if client.Room == nil {
		return
	}

	// Broadcast typing status to other users in the same room
	for _, roomClient := range client.Room.Clients {
		// Don't send to yourself
		if roomClient != client {
			typingMessage := Message{
				Type:      TypingMessage,
				Content:   "",
				Sender:    client.Email,
				Id:        generateId(),
				Timestamp: time.Now().Format(time.RFC3339),
				Room: Room{
					Id:   client.Room.Id,
					Name: client.Room.Name,
				},
			}

			if isTyping {
				typingMessage.Content = "is typing..."
			} else {
				typingMessage.Content = "stopped typing"
			}

			msgBytes, err := json.Marshal(typingMessage)
			if err != nil {
				log.Printf("Error marshalling typing message: %v\n", err)
				continue
			}

			if err := roomClient.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				log.Printf("Error sending typing message: %v", err)
			}
		}
	}
}
