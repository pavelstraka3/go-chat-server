package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"regexp"
	"sync"
)

type Client struct {
	Username string
	Conn     *websocket.Conn
	ChatRoom *ChatRoom
}

type ClientManager struct {
	Clients   map[string]*Client
	Usernames map[string]bool
	History   []string
	Rooms     map[string]*ChatRoom
	Lock      sync.Mutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		Clients:   make(map[string]*Client),
		Usernames: make(map[string]bool),
		History:   make([]string, 0),
		Rooms:     make(map[string]*ChatRoom),
	}
}

func (cm *ClientManager) AddClient(id string, client *Client) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	cm.Clients[id] = client
	log.Printf("Client %s - %s added. Total clients: %d\n", client.Username, id, len(cm.Clients))
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

func (cm *ClientManager) ReserveUsername(username string) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	cm.Usernames[username] = true
}

func (cm *ClientManager) ReleaseUsername(username string) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	delete(cm.Usernames, username)
}

func (cm *ClientManager) IsValidUsername(username string) (bool, string) {
	var validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	if username == "" {
		return false, "Username cannot be empty"
	}

	if len(username) < 3 || len(username) > 20 {
		return false, "Username must be between 3 and 20 characters"
	}

	if cm.Usernames[username] {
		return false, "Username already taken"
	}

	if !validUsernameRegex.MatchString(username) {
		return false, "Username can only contain letters, numbers, and underscores."
	}

	return true, ""
}

func (cm *ClientManager) FindClientByUsername(username string) *Client {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	for _, client := range cm.Clients {
		if client.Username == username {
			return client
		}
	}
	return nil
}

func (cm *ClientManager) GetOrCreateRoom(roomName string) *ChatRoom {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	room, exists := cm.Rooms[roomName]
	if !exists {
		// if the room doesn't exist, create it
		room = &ChatRoom{
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

func (cm *ClientManager) JoinRoom(roomName string, client *Client) {
	room := cm.GetOrCreateRoom(roomName)
	room.Clients[client.Username] = client
	client.ChatRoom = room
}

func (cm *ClientManager) BroadcastMessageToRoom(roomName string, message []byte, user string) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	room, exists := cm.Rooms[roomName]
	if !exists {
		log.Printf("Room %s does no exist", roomName)
		return
	}

	room.History = append(room.History, string(message))

	for _, client := range room.Clients {
		if err := sendMessage(client.Conn, ChatMessage, string(message), user); err != nil {
			log.Printf("Error sending message to client %s: %v\n", client.Username, err)
		}
	}
}
