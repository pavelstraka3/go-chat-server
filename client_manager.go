package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type Client struct {
	Email string
	Conn  *websocket.Conn
	Room  *Room
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

func (cm *ClientManager) JoinRoom(roomName string, client *Client) error {
	log.Printf("Attempting to join room: %s for client: %s", roomName, client.Email)
	dbRoom, err := getRoomByName(cm.Db, roomName)
	if err != nil {
		return fmt.Errorf("error getting room from database %s: %v", roomName, err)
	}

	if dbRoom == nil {
		newRoom, err := createRoom(cm.Db, roomName)
		if err != nil {
			return fmt.Errorf("error creating room %s: %v", roomName, err)
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

	log.Printf("Successfully joined room %s", roomName)
	return nil
}

func (cm *ClientManager) BroadcastMessageToRoom(roomName string, message []byte, user string) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()

	log.Printf("Broadcasting message to room %s: %s", roomName, string(message))

	room, exists := cm.Rooms[roomName]
	if !exists {
		log.Printf("Room %s does no exist", roomName)
		return
	}

	room.History = append(room.History, string(message))

	for _, client := range room.Clients {
		if err := sendMessage(client.Conn, RegularMessage, string(message), user, roomName); err != nil {
			log.Printf("Error sending message to client %s: %v\n", client.Email, err)
		}
	}
}
