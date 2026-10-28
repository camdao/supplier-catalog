package initialize

import (
	"catalog/global"
	"fmt"
)

func InitTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			supplier_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			price REAL NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := global.DB.Exec(query); err != nil {
		return fmt.Errorf("initialize products table: %w", err)
	}

	return nil
}
