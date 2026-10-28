package global

import (
	"catalog/pkg/setting"
	"database/sql"
)

var (
	Config setting.Config
	DB     *sql.DB
)
