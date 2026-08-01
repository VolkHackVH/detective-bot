package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is empty")
	}

	directory := filepath.Dir(path)

	if directory != "." {
		if err := os.MkdirAll(directory, 0755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// SQLite одновременно обслуживается через одно соединение.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
	}

	for _, query := range pragmas {
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			db.Close()
			return nil, fmt.Errorf("execute %q: %w", query, err)
		}
	}

	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	return db, nil
}
