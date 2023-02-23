package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
	"testing"
)

func assertHeader(t *testing.T, resp *http.Response, hkey string, hval string) {
	h := resp.Header.Get(hkey)
	assert.Equal(t, hval, h)
}

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

func HttpSetKey(data *TestSetKeyData, token string) *http.Response {
	req, err := http.NewRequest(http.MethodPost, buckets.getListenAddr()+"/"+data.bucket+"/"+data.key, bytes.NewBuffer(data.data))
	AddAuth(token, req)
	data.SetAliasHeader(req)

	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func HttpSearch(sk *TestSearchData, token string) *http.Response {
	var payload []byte
	req, err := http.NewRequest(http.MethodGet, buckets.getListenAddr()+"/"+sk.bucket, bytes.NewBuffer(payload))
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

type SearchResponseEntry struct {
	Key  string `json:"key"`
	Data string `json:"data,omitempty"`
}

func SearchResponseEntryFromResponse(resp *http.Response) []SearchResponseEntry {
	var result []SearchResponseEntry
	body, _ := ioutil.ReadAll(resp.Body)                  // response body is []byte
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	fmt.Println(string(body))
	fmt.Printf("Rec: %d\n", len(result))
	return result

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

type TestSetKeyData struct {
	bucket  string
	key     string
	data    []byte
	aliases []string
}

func NewTestSetKeyData(bucket string, key string, data []byte) *TestSetKeyData {
	return &TestSetKeyData{
		bucket:  bucket,
		key:     key,
		data:    data,
		aliases: make([]string, 0),
	}
}
func (tk *TestSetKeyData) AddAlias(alias string) {
	tk.aliases = append(tk.aliases, alias)
}

func (tk *TestSetKeyData) SetAliasHeader(req *http.Request) {
	if len(tk.aliases) > 0 {
		req.Header.Set(HEADER_ALIAS_KEY, strings.Join(tk.aliases, HEADER_ALIAS_SEPARATOR))
	}
}

type TestSearchData struct {
	bucket string
	prefix string
	max    int
	skip   int
}

func NewTestSearchData(bucket string, prefix string) *TestSearchData {
	return &TestSearchData{
		bucket: bucket,
		prefix: prefix,
		max:    math.MaxInt,
		skip:   0,
	}
}
