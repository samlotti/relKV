package main

import (
	"fmt"
	"os"
	"relKV/backup"
	"relKV/cmd"
	"relKV/commands"
)

const Version = "0.1.0"

const Banner = `
██████╗ ███████╗██╗     ██╗  ██╗██╗   ██╗
██╔══██╗██╔════╝██║     ██║ ██╔╝██║   ██║
██████╔╝█████╗  ██║     █████╔╝ ██║   ██║
██╔══██╗██╔══╝  ██║     ██╔═██╗ ╚██╗ ██╔╝
██║  ██║███████╗███████╗██║  ██╗ ╚████╔╝ 
╚═╝  ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  
                                         
`

func main() {

	if len(os.Args) > 1 {
		commands.ProcessCommands(os.Args[1:])
		return
	}

	fmt.Println(Banner)
	readyChannel := make(chan *cmd.BucketsDb)
	go cmd.BootServer(Version, readyChannel)
	bk := <-readyChannel

	if !cmd.EnvironmentInstance.GetBoolEnv("NOBACKUP") {
		backup.BackupsInit(cmd.BucketsInstance)
		go backup.BackupsInstance.Run()
	}

	bk.WaitTillStopped()

}
