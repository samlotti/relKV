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

func HttpStatus(token string) *http.Response {
	var payload []byte
	req, err := http.NewRequest(http.MethodGet, buckets.getListenAddr()+"/status", bytes.NewBuffer(payload))

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

	// Create a bucket not in the Environment list
	os.Mkdir("../test/data/testbucket", os.ModePerm)

	if len(testenv) == 0 {
		EnvironmentInstance.envFile = "../test/test.env"
	} else {
		EnvironmentInstance.envFile = testenv
	}

	readyChannel := make(chan *BucketsDb)
	go BootServer("test", readyChannel)
	<-readyChannel
	buckets.WaitTillStarted()

}
func stopTestServer() {
	buckets.shutDownServer()
	fmt.Printf("Server shut down\n")
}
