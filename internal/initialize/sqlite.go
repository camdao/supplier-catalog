package initialize

import (
	"catalog/global"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func InitSqlite() error {
	dsn := global.Config.Database.DSN

	if err := os.MkdirAll(filepath.Dir(dsn), 0755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open(global.Config.Database.Driver, dsn)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ping sqlite: %w", err)
	}

	global.DB = db
	return nil
}
