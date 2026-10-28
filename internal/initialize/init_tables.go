package initialize

import (
	"catalog/global"
	"fmt"
)

func InitTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS updates (
			id           TEXT PRIMARY KEY,
			status       TEXT NOT NULL DEFAULT 'in_progress',
			total_batches INTEGER NOT NULL DEFAULT 0,
			started_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS batches (
			id          TEXT NOT NULL,
			update_id   TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			error       TEXT NOT NULL DEFAULT '',
			processed_at DATETIME,
			PRIMARY KEY (update_id, id),
			FOREIGN KEY (update_id) REFERENCES updates(id)
		);

		CREATE TABLE IF NOT EXISTS products (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			price       INTEGER NOT NULL DEFAULT 0,
			category    TEXT NOT NULL DEFAULT '',
			version     INTEGER NOT NULL DEFAULT 0,
			withdrawn   INTEGER NOT NULL DEFAULT 0,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS changes (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			update_id   TEXT NOT NULL,
			batch_id    TEXT NOT NULL,
			product_id  TEXT NOT NULL,
			change_type TEXT NOT NULL,
			old_version INTEGER NOT NULL DEFAULT 0,
			new_version INTEGER NOT NULL DEFAULT 0,
			details     TEXT NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (update_id) REFERENCES updates(id)
		);
	`

	if _, err := global.DB.Exec(query); err != nil {
		return fmt.Errorf("initialize products table: %w", err)
	}

	return nil
}
