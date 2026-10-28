package initialize

import (
	"catalog/cmd"
	"fmt"
)

func Run() {
	fmt.Println("Loading configuration...")
	InitLoadConfig()
	fmt.Println("Configuration loaded")

	fmt.Println("Initializing SQLite...")
	InitSqlite()
	fmt.Println("SQLite initialized")

	fmt.Println("Initializing tables...")
	InitTables()
	fmt.Println("Tables initialized")

	fmt.Println("Running Cobra command...")
	cmd.Execute()
}
