package initialize

import "catalog/cmd"

func Run() {
	IniSqlite()
	cmd.Execute()
}
