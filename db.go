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
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}
}
