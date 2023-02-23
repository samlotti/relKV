package cmd

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCreateBucket(t *testing.T) {
	startTestServer("")

	resp := HttpCreateBucket("sample", buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Second good as well
	resp = HttpCreateBucket("sample", buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	_, err := os.ReadDir("../test/data/sample")
	assert.Nil(t, err)
	_, err = os.Stat("../test/data/sample/KEYREGISTRY")
	assert.Nil(t, err)

	// Should have a stats entry!
	assert.NotNil(t, stats.bucketStats["sample"])

	stopTestServer()
}

func TestCreateBucketBad(t *testing.T) {
	startTestServer("")

	resp := HttpCreateBucket("samp le", buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = HttpCreateBucket("status", buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

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

	// Secret not needed.
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

func Test_PostData1(t *testing.T) {
	startTestServer("")

	HttpCreateBucket("b1", buckets.authsecret.secret)

	// Secret not needed.
	data := NewTestSetKeyData("b1", "g1", []byte("{game1}"))
	data.AddAlias("p1:p2:g1")
	data.AddAlias("p2:p1:g1")
	resp := HttpSetKey(data, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// searchKeys
	sk := NewTestSearchData("b1", "")
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")

	rdata := SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 3, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "p1:p2:g1", rdata[1].Key)
	assert.Equal(t, "p2:p1:g1", rdata[2].Key)

	// --- get with the data
	sk = NewTestSearchData("b1", "")
	sk.values = true
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 3, len(rdata))
	assert.Equal(t, "g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)
	assert.Equal(t, "p1:p2:g1", rdata[1].Key)
	assert.Equal(t, "{game1}", rdata[1].Data)
	assert.Equal(t, "p2:p1:g1", rdata[2].Key)
	assert.Equal(t, "{game1}", rdata[2].Data)

	// --- get with the prefix
	sk = NewTestSearchData("b1", "")
	sk.values = true
	sk.prefix = "p1"
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)

	// --- get with the prefix -- explain
	sk = NewTestSearchData("b1", "")
	sk.values = true
	sk.prefix = "p1"
	sk.explain = true
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get all -- explain
	sk = NewTestSearchData("b1", "")
	sk.values = true
	sk.explain = true
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 3, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 3, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get skip -- explain
	sk = NewTestSearchData("b1", "")
	sk.values = true
	sk.explain = true
	sk.skip = 0
	sk.max = 1
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 2, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 0, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get skip -- explain
	sk = NewTestSearchData("b1", "")
	sk.values = true
	sk.explain = true
	sk.skip = 1
	sk.max = 1
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)

	assert.Equal(t, 3, stringToInt(resp.Header.Get("ex_row_read")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_selected")))
	assert.Equal(t, 1, stringToInt(resp.Header.Get("ex_rows_skipped")))

	// --- get skip -- explain
	sk = NewTestSearchData("b1", "")
	sk.values = true
	sk.explain = false
	sk.skip = 1
	sk.max = 1
	resp = HttpSearch(sk, buckets.authsecret.secret)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assertHeader(t, resp, RESP_HEADER_KVDB_FUNCTION, "searchKeys")
	rdata = SearchResponseEntryFromResponse(resp)
	assert.Equal(t, 1, len(rdata))
	assert.Equal(t, "p1:p2:g1", rdata[0].Key)
	assert.Equal(t, "{game1}", rdata[0].Data)

	stopTestServer()
}
