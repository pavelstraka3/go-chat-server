package main

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
)

func connectDB() *sql.DB {
	db, err := sql.Open("sqlite", "users.db")

	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error verifying database connection: %v", err)
	}
	return db
}

func createUserTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}
}

func creatRoomTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Error creating rooms table: %v", err)
	}
}

func createMessageTable(db *sql.DB) {
	query := `
		CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id INTEGER NOT NULL,
		sender TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (room_id) REFERENCES rooms (id)
	);`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Error creating rooms table: %v", err)
	}
}
