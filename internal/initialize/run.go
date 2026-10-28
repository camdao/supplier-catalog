package initialize

import "catalog/cmd"

func Run() {
	InitMysql()
	cmd.Execute()
}
