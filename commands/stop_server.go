package commands

import (
	"fmt"
	"log"
	"net"
	"relKV/cmd"
)

func ProcessCommands(cmds []string) {
	cmd.EnvInit()

	if len(cmds) > 1 {
		log.Fatal("Invalid command: ", cmds)
		return
	}

	switch cmds[0] {
	case "stop":
		handleStop()
	default:
		log.Fatal("Invalid command: ", cmds[0])
	}
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
	log.Printf("Server response:", string(buf[0:n]))
}
