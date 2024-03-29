package main

import (
	"fmt"
	"github.com/samlotti/relKV/backup"
	"github.com/samlotti/relKV/cmd"
	"github.com/samlotti/relKV/commands"
	"os"
)

const Version = "0.3.20"

const Banner = `
██████╗ ███████╗██╗     ██╗  ██╗██╗   ██╗
██╔══██╗██╔════╝██║     ██║ ██╔╝██║   ██║
██████╔╝█████╗  ██║     █████╔╝ ██║   ██║
██╔══██╗██╔══╝  ██║     ██╔═██╗ ╚██╗ ██╔╝
██║  ██║███████╗███████╗██║  ██╗ ╚████╔╝ 
╚═╝  ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  
./relKv help      - for help`

func main() {

	if len(os.Args) > 1 {
		commands.ProcessCommands(os.Args[1:])
		return
	}

	fmt.Println(Banner)
	readyChannel := make(chan *cmd.BucketsDb)
	go cmd.BootServer(Version, readyChannel)
	bk := <-readyChannel

	backup.ScpInit(cmd.BucketsInstance)

	if !cmd.EnvironmentInstance.GetBoolEnv("NOBACKUP") {
		backup.BackupsInit(cmd.BucketsInstance)
		go backup.BackupsInstance.Run()
	}

	bk.WaitTillStopped()

}
