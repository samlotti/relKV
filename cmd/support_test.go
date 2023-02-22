package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func AddAuth(token string, req *http.Request) {
	if len(token) == 0 {
		return
	}

	req.Header.Set("tkn", token)
}

func HttpCreateBucket(bname string, token string) *http.Response {
	var payload []byte
	req, err := http.NewRequest(http.MethodPut, buckets.getListenAddr()+"/"+bname, bytes.NewBuffer(payload))

	AddAuth(token, req)

	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func startTestServer(testenv string) {

	os.RemoveAll("../test/data")
	os.RemoveAll("../test/databk")

	os.Mkdir("../test/data", os.ModePerm)
	os.Mkdir("../test/databk", os.ModePerm)

	// Create a bucket not in the environment list
	os.Mkdir("../test/data/testbucket", os.ModePerm)

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
