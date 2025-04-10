package main

import (
	"database/sql"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerUser(db *sql.DB, email, password string) error {
	// hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `INSERT INTO users (email, password) VALUES (?, ?);`
	_, err = db.Exec(query, email, hashedPassword)
	if err != nil {
		return err
	}
	return nil
}

func loginUser(db *sql.DB, email, password string) (bool, error) {
	var hashedPassword string

	// retrieve the hashed password from the database
	query := `SELECT password FROM users WHERE email = ?`
	err := db.QueryRow(query, email).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // user does not exist
		}
		return false, err
	}

	// compare the hashed password with the provided password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return false, nil // invalid password
	}

	// Generate JWT token

	return true, nil
}

func getAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Email)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
