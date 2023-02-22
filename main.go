package main

import "kvDb/cmd"

const Version = "0.1.0"

func main() {
	readyChannel := make(chan bool)
	cmd.BootServer(Version, readyChannel)
}
