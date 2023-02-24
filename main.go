package main

import (
	"relKV/backup"
	"relKV/cmd"
)

const Version = "0.1.0"

func main() {
	readyChannel := make(chan *cmd.BucketsDb)
	go cmd.BootServer(Version, readyChannel)
	bk := <-readyChannel

	if !cmd.EnvironmentInstance.GetBoolEnv("NOBACKUP") {
		backup.BackupsInit(cmd.BucketsInstance)
		go backup.BackupsInstance.Run()
	}

	bk.WaitTillStopped()

}
