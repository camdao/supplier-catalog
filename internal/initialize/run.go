package initialize

import "catalog/cmd"

func Run() {
<<<<<<< HEAD
	IniSqlite()
=======
	InitMysql()
>>>>>>> 3c1ca2b47a1191b19daa42900de92ea59514dad2
	cmd.Execute()
}
