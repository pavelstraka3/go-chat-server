package main

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type Message struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	Sender    string      `json:"sender"`
	Id        string      `json:"id"`
	Room      Room        `json:"room,omitempty"`
	Target    string      `json:"target,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

func generateId() string {
	return uuid.New().String()
}

func sendMessage(conn *websocket.Conn, msgType MessageType, content string, user string, room string) error {
	message := Message{
		Type:      msgType,
		Content:   content,
		Sender:    user,
		Id:        generateId(),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if msgType != SystemMessage {
		message.Room = Room{
			ID:   0,
			Name: room,
		}
	}
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, msgBytes)
}

type MessageType string

const (
	RegularMessage MessageType = "regular"
	DirectMessage              = "direct"
	InvalidMessage             = "invalid"
	CommandMessage             = "command"
	SystemMessage              = "system"
)

type CommandType int

const (
	InvalidCommand CommandType = iota
	HelpCommand
	UsersCommand
	JoinCommand
)

type ParsedMessage struct {
	Type    MessageType
	Content string
	Target  string
	Command CommandType
	Room    Room
}

func parseMessage(rawMessage string) ParsedMessage {
	log.Printf("Parsing message: %s", rawMessage)
	// Attempt to parse the incoming JSON
	var message Message
	err := json.Unmarshal([]byte(rawMessage), &message)
	if err != nil {
		return ParsedMessage{
			Type:    InvalidMessage,
			Content: "Invalid message format. Ensure your message is valid JSON.",
		}
	}

	// check for message type and return parsed message
	switch message.Type {
	case DirectMessage:
		if message.Target == "" || message.Content == "" {
			return ParsedMessage{
				Type:    InvalidMessage,
				Content: "Invalid DM format.",
			}
		}
		return ParsedMessage{
			Type:    DirectMessage,
			Content: message.Content,
			Target:  message.Target,
		}
	case CommandMessage:
		switch message.Content {
		case "help":
			return ParsedMessage{
				Type:    CommandMessage,
				Content: "Available commands: /dm <username> <message> - Send a direct message\n /users - List of connected users",
				Command: HelpCommand,
			}
		case "users":
			return ParsedMessage{
				Type:    CommandMessage,
				Content: "user_list",
				Command: UsersCommand,
			}
		case "join":
			if message.Room.Name == "" {
				return ParsedMessage{
					Type:    InvalidMessage,
					Content: "Invalid room format. Use: {\"type\": \"command\", \"content\": \"join\", \"room\": \"roomName\"}",
				}
			}
			return ParsedMessage{
				Type:    CommandMessage,
				Command: JoinCommand,
				Content: message.Room.Name,
			}

		default:
			return ParsedMessage{
				Type:    InvalidMessage,
				Content: "Unknown command.",
			}
		}
	case RegularMessage:
		if message.Content == "" {
			return ParsedMessage{
				Type:    InvalidMessage,
				Content: "Chat message cannot be empty.",
			}
		}
		return ParsedMessage{
			Type:    RegularMessage,
			Content: message.Content,
			Room:    message.Room,
		}
	default:
		return ParsedMessage{
			Type:    InvalidMessage,
			Content: "Unknown message type.",
		}
	}
}

func saveMessageToDb(db *sql.DB, message ParsedMessage, roomId int, sender string) error {
	newId := generateId()

	query := `
	INSERT INTO messages 
	    (id, room_id, sender, content, date) 
	VALUES (?, ?, ?, ?, ?);`

	_, err := db.Exec(query, newId, roomId, sender, message.Content, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Printf("Error saving message to DB: %v", err)
		return err
	}

	log.Printf("Message saved to DB: %s", message.Content)
	return nil
}

func getMessages(db *sql.DB, roomId int, sender string) ([]Message, error) {
	query := "SELECT messages.id, messages.content, rooms.name, messages.sender, messages.date, room_id roomId FROM messages LEFT JOIN rooms ON messages.room_id = rooms.id WHERE room_id = ?"
	args := []interface{}{roomId}

	if sender != "" {
		query += " AND sender = ?"
		args = append(args, sender)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error retrieving messages from DB: %v", err)
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var id, content, room, user, date string
		if err := rows.Scan(&id, &content, &room, &user, &date); err != nil {
			log.Printf("Error scanning message row: %v", err)
			return nil, err
		}
		messages = append(messages, Message{
			Id:      id,
			Type:    RegularMessage,
			Content: content,
			Room: Room{
				ID:   roomId,
				Name: room,
			},
			Sender:    user,
			Timestamp: date,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over message rows: %v", err)
		return nil, err
	}

	return messages, nil
}
