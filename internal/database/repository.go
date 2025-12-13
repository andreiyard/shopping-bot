package database

import (
	"fmt"
	"time"
)

// Item represents a shopping list item
type Item struct {
	ID        int64
	ListID    string
	Name      string
	CreatedAt time.Time
	BoughtAt  *time.Time
	AddedBy   int64
	BoughtBy  *int64
}

// List represents a shopping list
type List struct {
	ID        string
	CreatedAt time.Time
	CreatedBy int64
}

// AddItem adds a new item to a shopping list
func (db *DB) AddItem(listID string, name string, addedBy int64) error {
	query := `INSERT INTO items (list_id, name, added_by) VALUES (?, ?, ?)`
	_, err := db.conn.Exec(query, listID, name, addedBy)
	if err != nil {
		return fmt.Errorf("failed to add item: %w", err)
	}
	return nil
}

// GetItems retrieves all unbought items for a list
func (db *DB) GetItems(listID string) ([]Item, error) {
	query := `
		SELECT id, list_id, name, created_at, bought_at, added_by, bought_by
		FROM items
		WHERE list_id = ? AND bought_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.ListID, &item.Name, &item.CreatedAt, &item.BoughtAt, &item.AddedBy, &item.BoughtBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return items, nil
}

// MarkBought marks an item as bought
func (db *DB) MarkBought(itemID int64, listID string, boughtBy int64) error {
	query := `
		UPDATE items
		SET bought_at = CURRENT_TIMESTAMP, bought_by = ?
		WHERE id = ? AND list_id = ? AND bought_at IS NULL
	`

	result, err := db.conn.Exec(query, boughtBy, itemID, listID)
	if err != nil {
		return fmt.Errorf("failed to mark item as bought: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item not found or already bought")
	}

	return nil
}

// GetHistory retrieves bought items for a list
func (db *DB) GetHistory(listID string, limit int) ([]Item, error) {
	query := `
		SELECT id, list_id, name, created_at, bought_at, added_by, bought_by
		FROM items
		WHERE list_id = ? AND bought_at IS NOT NULL
		ORDER BY bought_at DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, listID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.ListID, &item.Name, &item.CreatedAt, &item.BoughtAt, &item.AddedBy, &item.BoughtBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return items, nil
}

// DeleteItem deletes an item from the shopping list
func (db *DB) DeleteItem(itemID int64, listID string) error {
	query := `DELETE FROM items WHERE id = ? AND list_id = ?`

	result, err := db.conn.Exec(query, itemID, listID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item not found")
	}

	return nil
}

// === List Management ===

// CreateList creates a new shopping list
func (db *DB) CreateList(listID string, createdBy int64) error {
	query := `INSERT INTO lists (id, created_by) VALUES (?, ?)`
	_, err := db.conn.Exec(query, listID, createdBy)
	if err != nil {
		return fmt.Errorf("failed to create list: %w", err)
	}
	return nil
}

// ListExists checks if a list exists
func (db *DB) ListExists(listID string) (bool, error) {
	query := `SELECT COUNT(*) FROM lists WHERE id = ?`
	var count int
	err := db.conn.QueryRow(query, listID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check list existence: %w", err)
	}
	return count > 0, nil
}

// GetList retrieves a list by ID
func (db *DB) GetList(listID string) (*List, error) {
	query := `SELECT id, created_at, created_by FROM lists WHERE id = ?`

	var list List
	err := db.conn.QueryRow(query, listID).Scan(&list.ID, &list.CreatedAt, &list.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	return &list, nil
}

// === Session Management ===

// SetCurrentList sets the current list for a user
func (db *DB) SetCurrentList(userID int64, listID string) error {
	query := `
		INSERT INTO user_sessions (user_id, current_list_id, last_updated)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			current_list_id = excluded.current_list_id,
			last_updated = CURRENT_TIMESTAMP
	`
	_, err := db.conn.Exec(query, userID, listID)
	if err != nil {
		return fmt.Errorf("failed to set current list: %w", err)
	}
	return nil
}

// GetCurrentList gets the current list for a user
func (db *DB) GetCurrentList(userID int64) (string, error) {
	query := `SELECT current_list_id FROM user_sessions WHERE user_id = ?`

	var listID *string
	err := db.conn.QueryRow(query, userID).Scan(&listID)
	if err != nil {
		// No session found is not an error, return empty string
		return "", nil
	}

	if listID == nil {
		return "", nil
	}

	return *listID, nil
}
