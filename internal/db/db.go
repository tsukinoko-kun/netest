package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	conn *sql.DB
	mut  sync.Mutex
)

func init() {
	mut.Lock()
	defer mut.Unlock()

	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		panic(err)
	}
	dbPath := filepath.Join(dataDir, "history.db")
	conn, err = sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := setMode(ctx, conn); err != nil {
		panic(err)
	}

	err = migrate(ctx, conn)
	if err != nil {
		panic(err)
	}
}

func setMode(ctx context.Context, conn DBTX) error {
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL: %w", err)
	}
	return nil
}

//go:embed migrations/*.sql
var migrations embed.FS

func migrate(ctx context.Context, conn DBTX) error {
	activeMigrations, err := getActiveMigrations(conn, ctx)
	if err != nil {
		return fmt.Errorf("failed to get active migrations: %w", err)
	}

	migrationFiles, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	for _, migrationFile := range migrationFiles {
		name := migrationFile.Name()
		if slices.Contains(activeMigrations, name) {
			continue
		}
		migration, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("failed to read migration: %w", err)
		}
		_, err = conn.ExecContext(ctx, string(migration))
		if err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
		_, err = conn.ExecContext(ctx, "INSERT INTO migrations (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("failed to insert migration: %w", err)
		}
	}

	return nil
}

func getActiveMigrations(conn DBTX, ctx context.Context) ([]string, error) {
	conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY)")

	rows, err := conn.QueryContext(ctx, "SELECT name FROM migrations ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations: %w", err)
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, name)
	}
	return migrations, nil
}

func Direct() Querier {
	mut.Lock()
	defer mut.Unlock()

	if conn == nil {
		var err error
		conn, err = sql.Open("sqlite", filepath.Join(dataDir, "history.db"))
		if err != nil {
			panic(err)
		}
		if err := setMode(context.Background(), conn); err != nil {
			panic(err)
		}
	}
	return New(conn)
}

func Begin(ctx context.Context) (*TxQuerier, error) {
	mut.Lock()
	defer mut.Unlock()

	if conn == nil {
		var err error
		conn, err = sql.Open("sqlite", filepath.Join(dataDir, "history.db"))
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		if err := setMode(ctx, conn); err != nil {
			return nil, fmt.Errorf("failed to set database mode: %w", err)
		}
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	txQuerier := &TxQuerier{
		Queries: New(tx),
		tx:      tx,
	}
	return txQuerier, nil
}

func Close() error {
	if conn == nil {
		return nil
	}
	if err := conn.Close(); err != nil {
		conn = nil
		return fmt.Errorf("failed to close database: %w", err)
	}
	conn = nil
	return nil
}

type TxQuerier struct {
	*Queries
	tx *sql.Tx
}

func (txq *TxQuerier) Commit() error {
	return txq.tx.Commit()
}

func (txq *TxQuerier) Rollback() error {
	return txq.tx.Rollback()
}
