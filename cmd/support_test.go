package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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

func HttpDeleteKey(data *TestDeleteData, token string) *http.Response {
	var payload []byte
	req, err := http.NewRequest(http.MethodDelete, buckets.getListenAddr()+"/"+data.bucket+"/"+data.key, bytes.NewBuffer(payload))
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
	sk.setHeaders(req)

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
	Key   string `json:"key"`
	Data  string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
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
	bucket   string
	prefix   string
	max      int
	skip     int
	values   bool
	explain  bool
	b64      bool
	segments []string
}

func (d *TestSearchData) setHeaders(req *http.Request) {
	if d.values {
		req.Header.Set(HEADER_VALUES_KEY, "1")
	}
	if len(d.prefix) > 0 {
		req.Header.Set(HEADER_PREFIX_KEY, d.prefix)
	}
	if d.explain {
		req.Header.Set(HEADER_EXPLAIN_KEY, "1")
	}
	if d.b64 {
		req.Header.Set(HEADER_B64_KEY, "1")
	}
	if d.skip > 0 {
		req.Header.Set(HEADER_SKIP_KEY, fmt.Sprintf("%d", d.skip))
	}
	if d.max > 0 {
		req.Header.Set(HEADER_MAX_KEY, fmt.Sprintf("%d", d.max))
	}
	if len(d.segments) > 0 {
		req.Header.Set(HEADER_SEGMENT_KEY, strings.Join(d.segments, HEADER_SEGMENT_SEPARATOR))
	}

}

func (d *TestSearchData) addSegment(segment string) {
	d.segments = append(d.segments, segment)
}

func NewTestSearchData(bucket string, prefix string) *TestSearchData {
	return &TestSearchData{
		bucket: bucket,
		prefix: prefix,
		max:    0,
		skip:   0,
	}
}

func stringToInt(sval string) int {
	v, e := strconv.Atoi(sval)
	if e != nil {
		panic(e)
	}
	return v
}

func decodeB64(data string) string {
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func HttpListBuckets(token string) *http.Response {
	var payload []byte
	req, err := http.NewRequest(http.MethodGet, buckets.getListenAddr()+"/", bytes.NewBuffer(payload))
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

func ListBucketResponseEntryFromResponse(resp *http.Response) []BucketData {
	var result []BucketData
	body, _ := ioutil.ReadAll(resp.Body)                  // response body is []byte
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	fmt.Println(string(body))
	fmt.Printf("Rec: %d\n", len(result))
	return result

}

func HttpGetKeyValue(bucket string, key string, token string) *http.Response {
	var payload []byte
	req, err := http.NewRequest(http.MethodGet, buckets.getListenAddr()+"/"+bucket+"/"+key, bytes.NewBuffer(payload))
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

func ResponseBodyAsString(resp *http.Response) string {
	body, _ := ioutil.ReadAll(resp.Body) // response body is []byte
	fmt.Println(string(body))
	return string(body)

}

type TestDeleteData struct {
	bucket  string
	key     string
	aliases []string
}

func NewTestDeleteData(bucket string, key string) *TestDeleteData {
	return &TestDeleteData{
		bucket:  bucket,
		key:     key,
		aliases: make([]string, 0),
	}
}
func (tk *TestDeleteData) AddAlias(alias string) {
	tk.aliases = append(tk.aliases, alias)
}

func (tk *TestDeleteData) SetAliasHeader(req *http.Request) {
	if len(tk.aliases) > 0 {
		req.Header.Set(HEADER_ALIAS_KEY, strings.Join(tk.aliases, HEADER_ALIAS_SEPARATOR))
	}
}

type TestGetKeysData struct {
	bucket string
	keys   []string
	b64    bool
}

func (d *TestGetKeysData) setHeaders(req *http.Request) {
	if d.b64 {
		req.Header.Set(HEADER_B64_KEY, "1")
	}
}

func (d *TestGetKeysData) addKey(key string) {
	d.keys = append(d.keys, key)
}

func NewTestGetKeysData(bucket string) *TestGetKeysData {
	return &TestGetKeysData{
		bucket: bucket,
	}
}

func HttpGetKeys(sk *TestGetKeysData, secret string) *http.Response {
	var payload []byte
	var buf = bytes.NewBuffer(payload)
	for _, key := range sk.keys {
		buf.Write([]byte(key))
		buf.Write([]byte("\n"))
	}

	req, err := http.NewRequest(http.MethodPost, buckets.getListenAddr()+"/get/"+sk.bucket, buf)
	AddAuth(secret, req)
	sk.setHeaders(req)

	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp

}
