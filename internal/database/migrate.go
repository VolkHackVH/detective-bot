package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schema string

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply database schema: %w", err)
	}

	hasRejectionReason, err := columnExists(ctx, db, "evidence", "rejection_reason")
	if err != nil {
		return err
	}
	if !hasRejectionReason {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE evidence ADD COLUMN rejection_reason TEXT`,
		); err != nil {
			return fmt.Errorf("add evidence.rejection_reason: %w", err)
		}
	}

	return nil
}

func columnExists(
	ctx context.Context,
	db *sql.DB,
	table string,
	column string,
) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return false, fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			columnID     int
			name         string
			columnType   string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)

		if err := rows.Scan(
			&columnID,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		); err != nil {
			return false, fmt.Errorf("scan table %s metadata: %w", table, err)
		}

		if name == column {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate table %s metadata: %w", table, err)
	}

	return false, nil
}
