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
