package cmd

import (
	"fmt"
	"os"
	"testing"
)

func startTestServer(testenv string) {

	os.RemoveAll("../test/data")
	os.RemoveAll("../test/databk")

	os.Mkdir("../test/data", os.ModePerm)
	os.Mkdir("../test/databk", os.ModePerm)

	if len(testenv) == 0 {
		Environment.envFile = "../test/test.env"
	} else {
		Environment.envFile = testenv
	}
	Environment.logFile = ""

	readyChannel := make(chan bool)
	go BootServer("test", readyChannel)
	<-readyChannel
	buckets.waitTillStarted()

}
func stopTestServer() {
	buckets.shutDownServer()
	fmt.Printf("Server shut down\n")
}

func TestCreateBucket(t *testing.T) {
	startTestServer("")
	stopTestServer()
}
