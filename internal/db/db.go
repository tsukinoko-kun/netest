package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type HistoryEntry[T any] struct {
	ID    int64     `json:"id"`
	Value T         `json:"value"`
	Time  time.Time `json:"time"`
}

func New() (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory %s: %w", dataDir, err)
	}

	filePath := filepath.Join(dataDir, "history.db")
	conn, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database at %s: %w", filePath, err)
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create database tables: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	if db.conn != nil {
		if err := db.conn.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}
	return nil
}

func (db *DB) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS history_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		value_json TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history_entries(timestamp);
	`

	if _, err := db.conn.Exec(query); err != nil {
		return fmt.Errorf("failed to execute table creation query: %w", err)
	}

	return nil
}

func Insert[T any](db *DB, value T, timestamp time.Time) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value to JSON: %w", err)
	}

	// Store timestamp in UTC as RFC3339 format for consistent timezone handling
	timestampStr := timestamp.UTC().Format(time.RFC3339Nano)

	query := `INSERT INTO history_entries (value_json, timestamp) VALUES (?, ?)`
	if _, err := db.conn.Exec(query, string(valueJSON), timestampStr); err != nil {
		return fmt.Errorf("failed to insert history entry with timestamp %s: %w", timestampStr, err)
	}

	return nil
}

func RetrieveAll[T any](db *DB) ([]HistoryEntry[T], error) {
	query := `SELECT id, value_json, timestamp FROM history_entries ORDER BY timestamp ASC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query history entries: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry[T]
	for rows.Next() {
		var id int64
		var valueJSON string
		var timestampStr string

		if err := rows.Scan(&id, &valueJSON, &timestampStr); err != nil {
			return nil, fmt.Errorf("failed to scan history entry row: %w", err)
		}

		// Parse timestamp back from UTC
		timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp %s: %w", timestampStr, err)
		}

		var value T
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON value for entry ID %d: %w", id, err)
		}

		entries = append(entries, HistoryEntry[T]{
			ID:    id,
			Value: value,
			Time:  timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating over history entries: %w", err)
	}

	return entries, nil
}

func (db *DB) DeleteAll() error {
	query := `DELETE FROM history_entries`
	if _, err := db.conn.Exec(query); err != nil {
		return fmt.Errorf("failed to delete all history entries: %w", err)
	}
	return nil
}

func ReplaceAll[T any](db *DB, entries []HistoryEntry[T]) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for replacing all entries: %w", err)
	}
	defer tx.Rollback()

	// Delete all existing entries
	if _, err := tx.Exec(`DELETE FROM history_entries`); err != nil {
		return fmt.Errorf("failed to delete existing entries in transaction: %w", err)
	}

	// Insert new entries
	stmt, err := tx.Prepare(`INSERT INTO history_entries (value_json, timestamp) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement in transaction: %w", err)
	}
	defer stmt.Close()

	for i, entry := range entries {
		valueJSON, err := json.Marshal(entry.Value)
		if err != nil {
			return fmt.Errorf("failed to marshal value to JSON for entry %d: %w", i, err)
		}

		timestampStr := entry.Time.UTC().Format(time.RFC3339Nano)
		if _, err := stmt.Exec(string(valueJSON), timestampStr); err != nil {
			return fmt.Errorf("failed to insert entry %d with timestamp %s in transaction: %w", i, timestampStr, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for replacing all entries: %w", err)
	}

	return nil
}

func (db *DB) BeginTx() (*sql.Tx, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	return tx, nil
}

// Track adds a new entry to the database
func Track[T any](db *DB, value T) error {
	now := time.Now()
	if err := Insert(db, value, now); err != nil {
		return fmt.Errorf("failed to track entry at %s: %w", now.Format(time.RFC3339), err)
	}
	return nil
}

// Summarize groups history entries by time periods and stores them transactionally
func Summarize[T any](db *DB, join func(entries []HistoryEntry[T]) HistoryEntry[T]) error {
	entries, err := RetrieveAll[T](db)
	if err != nil {
		return fmt.Errorf("failed to retrieve entries for summarization: %w", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastMonth := now.AddDate(0, -1, 0)
	threeMonthsAgo := now.AddDate(0, -3, 0)

	var result []HistoryEntry[T]
	var currentGroup []HistoryEntry[T]
	var currentGroupKey string

	for _, entry := range entries {
		// Skip entries older than 3 months - they should already be summarized
		if entry.Time.Before(threeMonthsAgo) {
			result = append(result, entry)
			continue
		}

		// Current day entries - don't touch
		if entry.Time.After(today) || entry.Time.Equal(today) {
			result = append(result, entry)
			continue
		}

		var groupKey string
		if entry.Time.After(lastMonth) {
			// Last month: group by hour
			groupKey = fmt.Sprintf("%d-%02d-%02d-%02d",
				entry.Time.Year(), entry.Time.Month(), entry.Time.Day(), entry.Time.Hour())
		} else {
			// Older than last month: group by 6-hour periods
			sixHourPeriod := entry.Time.Hour() / 6
			groupKey = fmt.Sprintf("%d-%02d-%02d-%d",
				entry.Time.Year(), entry.Time.Month(), entry.Time.Day(), sixHourPeriod)
		}

		// If this is a new group, process the previous group
		if currentGroupKey != "" && currentGroupKey != groupKey {
			if len(currentGroup) > 1 {
				result = append(result, join(currentGroup))
			} else if len(currentGroup) == 1 {
				result = append(result, currentGroup[0])
			}
			currentGroup = nil
		}

		currentGroup = append(currentGroup, entry)
		currentGroupKey = groupKey
	}

	// Process the last group
	if len(currentGroup) > 1 {
		result = append(result, join(currentGroup))
	} else if len(currentGroup) == 1 {
		result = append(result, currentGroup[0])
	}

	// Store the summarized entries using transaction
	if err := ReplaceAll(db, result); err != nil {
		return fmt.Errorf("failed to store summarized entries: %w", err)
	}

	return nil
}

// ExtractValue extracts the value from a history entry
func ExtractValue[T any](entry HistoryEntry[T]) T {
	return entry.Value
}

// ExtractTime extracts the time from a history entry
func ExtractTime[T any](entry HistoryEntry[T]) time.Time {
	return entry.Time
}

// Unpack extracts values and median time from a slice of history entries
func Unpack[T any](entries []HistoryEntry[T]) ([]T, time.Time) {
	result := make([]T, len(entries))
	for i, e := range entries {
		result[i] = e.Value
	}
	return result, entries[len(entries)/2].Time
}
