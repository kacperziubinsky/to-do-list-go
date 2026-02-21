package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) error {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("database open error: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	if err = createTables(); err != nil {
		return fmt.Errorf("table creation error: %w", err)
	}

	log.Println("Connected to SQLite database!")
	return nil
}

func createTables() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);`
	if _, err := DB.Exec(usersTable); err != nil {
		return fmt.Errorf("users table: %w", err)
	}

	tasksTable := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		description TEXT,
		status TEXT,
		date TEXT,
		user_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`
	if _, err := DB.Exec(tasksTable); err != nil {
		return fmt.Errorf("tasks table: %w", err)
	}

	return nil
}