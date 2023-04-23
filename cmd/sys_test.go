package cmd

import (
	. "github.com/samlotti/relKV/common"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCreateBucket(t *testing.T) {
	startTestServer("")

	resp := HttpCreateBucket("sample", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Second good as well
	resp = HttpCreateBucket("sample", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	_, err := os.ReadDir("../test/data/sample")
	assert.Nil(t, err)
	_, err = os.Stat("../test/data/sample/KEYREGISTRY")
	assert.Nil(t, err)

	// Should have a StatsInstance entry!
	assert.NotNil(t, StatsInstance.bucketStats["sample"])

	stopTestServer()
}

func TestCreateBucketBad(t *testing.T) {
	startTestServer("")

	resp := HttpCreateBucket("samp le", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = HttpCreateBucket("status", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = HttpCreateBucket("Test", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = HttpCreateBucket(" test1 ", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = HttpCreateBucket("te st1", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = HttpCreateBucket("te,et1", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// This one fails with match of path
	resp = HttpCreateBucket("te/et1", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	assert.False(t, validateBucketName("te/et1"))

	resp = HttpCreateBucket("teet1", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	stopTestServer()
}

func TestCreateBucket_StatusUnauthorized(t *testing.T) {
	startTestServer("")

	resp := HttpCreateBucket("sample", "bad secret")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	stopTestServer()
}

func Test_StatusCall(t *testing.T) {
	startTestServer("")

	resp := HttpStatus("bad secret")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	assert.True(t, strings.Contains(string(body), "Current time"))
	assert.True(t, strings.Contains(string(body), "Writes"))
	assert.True(t, strings.Contains(string(body), "ctl_games"))

	stopTestServer()
}

func Test_PostData1_invalid(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	//data := NewTestSetKeyData("b1", "g1\n", []byte("{game1}"))
	//resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	//defer resp.Body.Close()
	//assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Cant actually post them
	assert.False(t, isKeyValid("test\n"))
	assert.False(t, isKeyValid("test\r"))

	stopTestServer()
}

func Test_PostData1(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// searchKeys
	sk := NewTestSearchData("b1")
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata := SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 3, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "p1:p2:g1", rdata[1].Key)
	assert.Equal(t, "p2:p1:g1", rdata[2].Key)

	// --- get with the data
	sk = NewTestSearchData("b1")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 3, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)
	assert.Equal(t, "p1:p2:g1", rdata[1].Key)
	assert.Equal(t, "{game1}", rdata[1].Data)
	assert.Equal(t, "p2:p1:g1", rdata[2].Key)
	assert.Equal(t, "{game1}", rdata[2].Data)

	// --- get with the prefix
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.prefix = "p1"
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)

	// --- get with the prefix -- explain
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.prefix = "p1"
	sk.explain = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get all -- explain
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.explain = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 3, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 3, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get skip -- explain
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.explain = true
	sk.skip = 0
	sk.max = 1
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 2, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get skip -- explain
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.explain = true
	sk.skip = 1
	sk.max = 1
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 3, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get skip
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.explain = false
	sk.skip = 1
	sk.max = 1
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)

	// --- get skip -- b64
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.explain = false
	sk.skip = 1
	sk.b64 = true
	sk.max = 1
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "{game1}", decodeB64(rdata[0].Data))

	// --- get non alias
	sk = NewTestSearchData("b1")
	sk.values = true
	sk.b64 = true
	sk.prefix = "g1"
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1}", decodeB64(rdata[0].Data))

	// --- search bad bucket
	sk = NewTestSearchData("b1dd")
	sk.values = true
	sk.b64 = true
	sk.prefix = "g1"
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	stopTestServer()
}

func Test_ListBuckets(t *testing.T) {
	startTestServer("")

	resp := HttpListBuckets(BucketsInstance.authsecret.secret)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()
	bucketData := ListBucketResponseEntryFromResponse(resp)

	assert.Equal(t, 3, len(bucketData))

	names := map[string]string{
		"ctl_games":  "x",
		"ct_games":   "x",
		"testbucket": "x",
	}
	for _, data := range bucketData {
		assert.True(t, names[data.Name] != "")
		delete(names, data.Name)
	}
	assert.Equal(t, 0, len(names))
	stopTestServer()
}

func Test_GetKeyValue(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get alias entry
	resp = HttpGetKeyValue("b1", "p1:p2:g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	body := ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", body)

	// Get regular entry
	resp = HttpGetKeyValue("b1", "g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	body = ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", body)

	// Get not found entry
	resp = HttpGetKeyValue("b1", "g1xxx", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()

	// Get not found entry  bucket!
	resp = HttpGetKeyValue("b1x", "g1xxx", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	defer resp.Body.Close()

	stopTestServer()
}

func Test_Delete1(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get alias entry
	resp = HttpGetKeyValue("b1", "p1:p2:g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	body := ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", body)

	// Get regular entry
	resp = HttpGetKeyValue("b1", "g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	body = ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", body)

	// Delete
	td := NewTestDeleteData("b1", "g1")
	td.AddAlias("p1:p2:g1")
	td.AddAlias("p2:p1:g1")
	resp = HttpDeleteKey(td, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, "rec_deleted", "3")

	// Get not found entry
	resp = HttpGetKeyValue("b1", "g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()

	resp = HttpGetKeyValue("b1", "p2:p1:g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()

	stopTestServer()
}

func Test_Delete12_dont_delete_aliases(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get alias entry
	resp = HttpGetKeyValue("b1", "p1:p2:g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	body := ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", body)

	// Get regular entry
	resp = HttpGetKeyValue("b1", "g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	body = ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", body)

	// Delete
	td := NewTestDeleteData("b1", "g1")
	//td.AddAlias("p1:p2:g1")
	//td.AddAlias("p2:p1:g1")
	resp = HttpDeleteKey(td, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	ResponseBodyAsString(resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, "rec_deleted", "1")

	// Get not found entry
	resp = HttpGetKeyValue("b1", "g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()

	// Still not found. But takes space in DB
	resp = HttpGetKeyValue("b1", "p2:p1:g1", BucketsInstance.authsecret.secret)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()

	stopTestServer()
}

func Test_PostData1_no_aliases(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// searchKeys
	sk := NewTestSearchData("b1")
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata := SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)

	// --- get with the data
	sk = NewTestSearchData("b1")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)

	stopTestServer()
}

func Test_PostData1_segment_search(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g2", []byte("{game2}"))
	data.AddAlias("p1:p3:g2")
	data.AddAlias("p3:p1:g2")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g3", []byte("{game3}"))
	data.AddAlias("p1:p4:g3")
	data.AddAlias("p4:p1:g3")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g4", []byte("{game4}"))
	data.AddAlias("p1:p2:g4")
	data.AddAlias("p2:p1:g4")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// searchKeys - for player 1, should get 4 games
	sk := NewTestSearchData("b1")
	sk.prefix = "p1"
	sk.addSegment("p1") // <- since prefix, this is not needed??? in this case
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata := SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 4, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "p1:p2:g4", rdata[1].Key)
	assert.Equal(t, "p1:p3:g2", rdata[2].Key)
	assert.Equal(t, "p1:p4:g3", rdata[3].Key)

	// --- do an explain
	sk.explain = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	assert.Equal(t, 4, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 4, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// searchKeys - for player 1 and p2, should get 2 games
	sk = NewTestSearchData("b1")
	sk.prefix = "p1"
	sk.addSegment("p1") // <- since prefix, this is not needed??? in this case
	sk.addSegment("p2") // <- since prefix, this is not needed??? in this case
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 2, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "p1:p2:g4", rdata[1].Key)

	// --- do an explain
	sk.explain = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	assert.Equal(t, 4, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 2, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// searchKeys - for player 1 and p2, with explicit prefix, should get 2 games
	sk = NewTestSearchData("b1")
	sk.prefix = "p1:p2"
	sk.addSegment("p1") // <- since prefix, this is not needed??? in this case
	sk.addSegment("p2")

	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 2, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "p1:p2:g4", rdata[1].Key)

	// --- do an explain
	sk.explain = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	assert.Equal(t, 2, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 2, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// Test blank segments

	// searchKeys - for player 1 and p2, with explicit prefix, should get 2 games
	sk = NewTestSearchData("b1")
	sk.prefix = "p1:p2"
	sk.addSegment("p1") // <- since prefix, this is not needed??? in this case
	sk.addSegment("")
	sk.addSegment("p2")
	sk.addSegment("")

	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 2, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "p1:p2:g4", rdata[1].Key)

	stopTestServer()
}

func Test_PostData1_get_key_post(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g2", []byte("{game2}"))
	data.AddAlias("p1:p3:g2")
	data.AddAlias("p3:p1:g2")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g3", []byte("{game3}"))
	data.AddAlias("p1:p4:g3")
	data.AddAlias("p4:p1:g3")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g4", []byte("{game4}"))
	data.AddAlias("p1:p2:g4")
	data.AddAlias("p2:p1:g4")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// searchKeys - for player 1, should get 4 games
	sk := NewTestGetKeysData("b1")
	sk.addKey("g1")
	sk.addKey("g2")
	sk.addKey("g3")
	sk.addKey("g4")
	resp = HttpGetKeys(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "getKeys")

	rdata := SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 4, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)
	assert.Equal(t, "g2", rdata[1].Key)
	assert.Equal(t, "{game2}", rdata[1].Data)
	assert.Equal(t, "g3", rdata[2].Key)
	assert.Equal(t, "{game3}", rdata[2].Data)
	assert.Equal(t, "g4", rdata[3].Key)
	assert.Equal(t, "{game4}", rdata[3].Data)

	// =============================================
	// searchKeys - for aliases
	sk = NewTestGetKeysData("b1")
	sk.addKey("p1:p2:g4")
	sk.addKey("p3:p1:g2")
	resp = HttpGetKeys(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "getKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 2, len(rdata))
	assert.Equal(t, "p1:p2:g4", rdata[0].Key)
	assert.Equal(t, "{game4}", rdata[0].Data)
	assert.Equal(t, "p3:p1:g2", rdata[1].Key)
	assert.Equal(t, "{game2}", rdata[1].Data)

	// =============================================
	// searchKeys - for windowed aliases
	td := NewTestDeleteData("b1", "g2")
	resp = HttpDeleteKey(td, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()

	sk = NewTestGetKeysData("b1")
	sk.addKey("g1")
	sk.addKey("g2") // <-- bad
	sk.addKey("p1:p2:g4")
	sk.addKey("p3:p1:g2") // <-- bad
	resp = HttpGetKeys(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "getKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 4, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)

	assert.Equal(t, "g2", rdata[1].Key)
	assert.Equal(t, "Key not found", rdata[1].Error)

	assert.Equal(t, "p1:p2:g4", rdata[2].Key)
	assert.Equal(t, "{game4}", rdata[2].Data)

	assert.Equal(t, "p3:p1:g2", rdata[3].Key)
	assert.Equal(t, "Key not found", rdata[3].Error)

	// =============================================
	// searchKeys - for windowed aliases - base64
	td = NewTestDeleteData("b1", "g2")
	resp = HttpDeleteKey(td, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()

	sk = NewTestGetKeysData("b1")
	sk.addKey("g1")
	sk.addKey("g2") // <-- bad
	sk.addKey("p1:p2:g4")
	sk.addKey("p3:p1:g2") // <-- bad
	sk.b64 = true
	resp = HttpGetKeys(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "getKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 4, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "e2dhbWUxfQ==", rdata[0].Data)

	assert.Equal(t, "g2", rdata[1].Key)
	assert.Equal(t, "Key not found", rdata[1].Error)

	assert.Equal(t, "p1:p2:g4", rdata[2].Key)
	assert.Equal(t, "e2dhbWU0fQ==", rdata[2].Data)

	assert.Equal(t, "p3:p1:g2", rdata[3].Key)
	assert.Equal(t, "Key not found", rdata[3].Error)

	stopTestServer()
}

func Test_PostData_duplicate_test1(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	data.AddAlias("g1alias")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g2", []byte("{game2}"))
	data.AddAlias("p1:p3:g2")
	data.AddAlias("p3:p1:g2")
	data.AddAlias("g1alias")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_DUPLICATE_ERROR, "g1alias")

	// --- get skip -- b64
	sk := NewTestSearchData("b1")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata := SearchResponseEntryFromResponse(resp)

	// Should only have the 4 records
	assert.Equal(t, 4, len(rdata))
	// {"key":"g1alias","value":"{game1}"},
	assert.Equal(t, "g1alias", rdata[1].Key)
	assert.Equal(t, "{game1}", rdata[1].Data)

	stopTestServer()
}

// Test_PostData_duplicate_test2 - test alias try to override a non alias
func Test_PostData_duplicate_test2(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	data.AddAlias("g1alias")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g2", []byte("{game2}"))
	data.AddAlias("p1:p3:g2")
	data.AddAlias("p3:p1:g2")
	data.AddAlias("g1") // << yikes!!!
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_DUPLICATE_ERROR, "g1")

	// --- get skip -- b64
	sk := NewTestSearchData("b1")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata := SearchResponseEntryFromResponse(resp)

	// Should only have the 4 records
	assert.Equal(t, 4, len(rdata))
	// {"key":"g1alias","value":"{game1}"},
	assert.Equal(t, "g1alias", rdata[1].Key)
	assert.Equal(t, "{game1}", rdata[1].Data)

	stopTestServer()
}

func Test_PostData1_segment_search_with_paths(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "games/g1:45", []byte("{game1}"))
	data.AddAlias("games/p1:p2:g1")
	data.AddAlias("games/p2:p1:g1")
	data.AddAlias("games/p5:p6:g1")
	data.AddAlias("games/p5:p7:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// searchKeys - for player 1, should get 4 games
	sk := NewTestSearchData("b1")
	sk.prefix = "games"
	sk.addSegment("p2")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata := SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 2, len(rdata))
	assert.Equal(t, "games/p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "games/p2:p1:g1", rdata[1].Key)

	// searchKeys - for player 1 and p2, should get 2 games
	sk = NewTestSearchData("b1")
	sk.prefix = "games/"
	sk.addSegment("p1") // <- since prefix, this is not needed??? in this case
	sk.addSegment("p2") // <- since prefix, this is not needed??? in this case
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")

	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 2, len(rdata))
	assert.Equal(t, "games/p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "games/p2:p1:g1", rdata[1].Key)

	// searchKeys - for player 1 and p2, should get 2 games
	resp = HttpGetKeyValue("b1", "games/g1:45", BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "getKey")

	rstr := ResponseBodyAsString(resp)
	assert.Equal(t, "{game1}", rstr)

	stopTestServer()
}

// Test_PostData_duplicate_test2 - test alias try to override a non alias
func Test_PostData_update(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g1", []byte("{game1b}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	sk := NewTestSearchData("b1")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata := SearchResponseEntryFromResponse(resp)

	// Should only have the 4 records
	assert.Equal(t, 3, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1b}", rdata[0].Data)
	assert.Equal(t, "p1:p2:g1", rdata[1].Key)
	assert.Equal(t, "{game1b}", rdata[1].Data)

	stopTestServer()
}

// Test_PostData_duplicate_test2 - test alias try to override a non alias
func Test_PostData_update_alias_autoupdate(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", BucketsInstance.authsecret.secret)

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	data = NewTestSetKeyData("b1", "g1", []byte("{game1b}"))
	//data.AddAlias("p1:p2:g1")
	//data.AddAlias("p2:p1:g1")
	resp = HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	sk := NewTestSearchData("b1")
	sk.values = true
	resp = HttpSearch(sk, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_RELDB_FUNCTION, "searchKeys")
	rdata := SearchResponseEntryFromResponse(resp)

	// Should only have the 4 records
	assert.Equal(t, 3, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1b}", rdata[0].Data)
	assert.Equal(t, "p1:p2:g1", rdata[1].Key)
	assert.Equal(t, "{game1b}", rdata[1].Data)

	stopTestServer()
}

func Test_PostData_wrong_bucket(t *testing.T) {
	startTestServer("")

	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, BucketsInstance.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_ERROR_MSG, "bucket not found")

	stopTestServer()
}
