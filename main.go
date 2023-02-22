package main

import "kvDb/cmd"

const Version = "0.1.0"

func main() {
	readyChannel := make(chan *cmd.BucketsDb)
	go cmd.BootServer(Version, readyChannel)
	bk := <-readyChannel
	bk.WaitTillStopped()

}
