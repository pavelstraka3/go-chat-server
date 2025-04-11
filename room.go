package main

import (
	"database/sql"
)

type Room struct {
	ID        int
	Name      string
	CreatedAt string
	Clients   map[string]*Client
	History   []string
}

func createRoom(db *sql.DB, name string) (*Room, error) {
	query := `
	INSERT INTO rooms (name) VALUES (?);
	`
	_, err := db.Exec(query, name)

	if err != nil {
		return nil, err
	}
	room, err := getRoomByName(db, name)

	if err != nil {
		return nil, err
	}

	room.Clients = make(map[string]*Client)
	room.History = make([]string, 0)
	return room, nil
}

func getRoomByName(db *sql.DB, roomName string) (*Room, error) {
	query := "SELECT id, name, created_at FROM rooms WHERE name = ?"
	row := db.QueryRow(query, roomName)

	var room Room
	err := row.Scan(&room.ID, &room.Name, &room.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	room.Clients = make(map[string]*Client)
	room.History = make([]string, 0)
	return &room, nil
}

func getAllRooms(db *sql.DB) ([]Room, error) {
	query := "SELECT id, name, created_at FROM rooms"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		var room Room
		if err := rows.Scan(&room.ID, &room.Name, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}
