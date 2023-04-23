package commands

import (
	"fmt"
	"github.com/samlotti/relKV/cmd"
	"log"
	"net"
)

func ProcessCommands(cmds []string) {
	cmd.EnvInit()

	//if len(cmds) > 1 {
	//	log.Fatal("Invalid command: ", cmds)
	//	return
	//}

	switch cmds[0] {
	case "help":
		handleHelp()
	case "stop":
		handleStop()
	case "restore":
		handleRestore(cmds)
	default:
		log.Fatal("Invalid command: ", cmds[0])
		handleHelp()
	}
}

func handleHelp() {
	fmt.Println("Commands are: ")
	fmt.Println(" stop -> stop the running instance ")
	fmt.Println(" restore -> restore a backup file ")
	fmt.Println("     restore {backupfilename} {databaseName}")

}

func handleStop() {

	unixSocket := cmd.EnvironmentInstance.GetEnv("CMD_UNIX_SOCKET", "")
	if len(unixSocket) == 0 {
		fmt.Println("No socket defined in the environment variable: CMD_UNIX_SOCKET")
	}

	c, err := net.Dial("unix", unixSocket)
	if err != nil {
		log.Fatal("cannot connect to server:", err.Error())
	}
	defer c.Close()

	c.Write([]byte("stop\n"))

	buf := make([]byte, 1024)
	n, err := c.Read(buf[:])
	if err != nil {
		log.Fatal("server disconnected")
	}
	log.Printf("Server response: %s", string(buf[0:n]))
}
