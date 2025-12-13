package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

// Open creates a new database connection and initializes the schema
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// Create tables if they don't exist
	if err := db.createTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// createTables initializes the database schema
func (db *DB) createTables() error {
	schema := `
	-- Shopping lists table
	CREATE TABLE IF NOT EXISTS lists (
		id TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by INTEGER NOT NULL
	);

	-- User sessions table (tracks which list each user is currently using)
	CREATE TABLE IF NOT EXISTS user_sessions (
		user_id INTEGER PRIMARY KEY,
		current_list_id TEXT,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (current_list_id) REFERENCES lists(id) ON DELETE SET NULL
	);

	-- Shopping items table
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		bought_at DATETIME,
		added_by INTEGER NOT NULL,
		bought_by INTEGER,
		FOREIGN KEY (list_id) REFERENCES lists(id) ON DELETE CASCADE
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_list_id ON items(list_id);
	CREATE INDEX IF NOT EXISTS idx_bought_at ON items(bought_at);
	CREATE INDEX IF NOT EXISTS idx_added_by ON items(added_by);
	CREATE INDEX IF NOT EXISTS idx_current_list ON user_sessions(current_list_id);
	CREATE INDEX IF NOT EXISTS idx_created_by ON lists(created_by);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}
