package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"strings"
)

const (
	SystemMessage = "system"
	ChatMessage   = "chat"
)

type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func sendMessage(conn *websocket.Conn, msgType, content string) error {
	message := Message{
		Type:    msgType,
		Content: content,
	}
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, msgBytes)
}

type MessageType int

const (
	RegularMessage MessageType = iota
	DirectMessage
	InvalidMessage
	CommandMessage
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
}

func parseMessage(message string) ParsedMessage {
	// check for direct message pattern
	if strings.HasPrefix(message, "/dm") {
		parts := strings.SplitN(string(message), " ", 3)

		if len(parts) < 3 {
			return ParsedMessage{
				Type:    InvalidMessage,
				Content: "Invalid DM format. Use: /dm <username> <message>",
			}
		}

		return ParsedMessage{
			Type:    DirectMessage,
			Content: parts[2],
			Target:  parts[1],
		}
	}

	// check for system commands
	if strings.HasPrefix(message, "/help") {
		return ParsedMessage{
			Type:    CommandMessage,
			Content: "Available commands: /dm <username> <message> - Send a direct message\n /users - List of connected users",
			Command: HelpCommand,
		}
	}

	// list of users
	if strings.HasPrefix(message, "/users") {
		return ParsedMessage{
			Type:    CommandMessage,
			Content: "user_list",
			Command: UsersCommand,
		}
	}

	// join room
	if strings.HasPrefix(message, "/join") {
		parts := strings.SplitN(message, " ", 2)
		if len(parts) < 2 {
			return ParsedMessage{
				Type:    InvalidMessage,
				Content: "Invalid room format. Use: /join <roomName>",
			}
		}

		return ParsedMessage{
			Type:    CommandMessage,
			Command: JoinCommand,
			Content: parts[1],
		}
	}

	// Otherwise it's regular message
	return ParsedMessage{
		Type:    RegularMessage,
		Content: message,
	}
}
