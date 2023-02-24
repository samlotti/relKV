package cmd

import (
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	. "relKV/common"
	"strconv"
	"strings"
)

type Environment struct {
	envFile string
}

var EnvironmentInstance Environment = Environment{
	envFile: ".env",
}

// EnvInit - Called at startup.
func EnvInit() {
	var fileName = EnvironmentInstance.envFile
	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

// GetEnv -- Gets the value of the Environment.
// If not specified and no default in the .env file it will return fallback
func (e *Environment) GetEnv(key string, fallback string) string {
	val, found := os.LookupEnv(key)
	if !found {
		val, found = viper.Get(key).(string)
	}
	if !found {
		return fallback
	}
	return val
}

// GetBoolEnv - reads an Environment variable and converts to a boolean.
// 	true values are:   "1", "t", "T", "true", "TRUE", "True"
//	false values are:  "0", "f", "F", "false", "FALSE", "False"
//  any other value will panic with an appropriate message
func (e *Environment) GetBoolEnv(key string) bool {
	val := e.GetEnv(key, "f")
	bval, err := strconv.ParseBool(val)
	if err != nil {
		panic(fmt.Sprintf("EnvironmentInstance variable invalid format: %s is expected to be a bool, found:%s", key, val))
	}
	return bval
}

func (e *Environment) LookupEnv(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		val, ok = viper.Get(key).(string)
	}
	return val, ok
}

func (e *Environment) GetBucketArray(key string) []BucketName {
	val := e.GetEnv(key, "")
	if len(val) == 0 {
		log.Fatal(fmt.Sprintf("%s not defined in Environment", key))
	}

	r := make([]BucketName, 0)
	for _, bname := range strings.Split(val, ",") {
		if !validateBucketName(string(bname)) {
			panic(fmt.Sprintf("bad bucket name %s", bname))
		}

		r = append(r, BucketName(strings.TrimSpace(bname)))
	}
	return r

}

func (e *Environment) GetIntArray(key string) []int {
	val := e.GetEnv(key, "")
	if len(val) == 0 {
		log.Fatal(fmt.Sprintf("%s not defined in Environment", key))
	}

	r := make([]int, 0)
	for _, bname := range strings.Split(val, ",") {
		val, err := strconv.Atoi(strings.TrimSpace(bname))
		if err != nil {
			log.Fatal(fmt.Sprintf("invalid list of integers for %s found %s", key, bname))
		}
		r = append(r, val)
	}
	return r

}

func getKeyByte(r *http.Request) []byte {
	vars := mux.Vars(r)
	return []byte(vars["key"])
}

func isKeyValid(key string) bool {
	if strings.Contains(key, "\n") {
		return false
	}
	if strings.Contains(key, "\r") {
		return false
	}
	return true
}

func getHeaderKey(key string, r *http.Request) string {
	data := r.URL.Query().Get(key)
	if len(data) == 0 {
		data = r.Header.Get(key)
	}
	return data
}

func getHeaderKeyBool(key string, r *http.Request) bool {
	data := getHeaderKey(key, r)
	if data != "" {
		return data == "1"
	}
	return false
}

func getHeaderKeyInt(key string, dflt int, r *http.Request) int {
	data := getHeaderKey(key, r)
	if data == "" {
		return dflt
	}
	val, err := strconv.Atoi(data)
	if err != nil {
		panic(fmt.Errorf("invalid value for %s, expected int found: %s", key, data))
	}
	return val
}

func getSegments(segmentsArg string) []string {
	var segments []string
	if len(segmentsArg) > 0 {
		for _, segment := range strings.Split(segmentsArg, HEADER_SEGMENT_SEPARATOR) {
			if len(segment) == 0 {
				continue
			}
			// Do this once instead of on each check
			segments = append(segments, HEADER_SEGMENT_SEPARATOR+segment+HEADER_SEGMENT_SEPARATOR)
		}
	}
	return segments
}

// segmentMatch - return true if the contains all the segments
// expects all segments to start and end with :
// so :game1234:user1:user2:user3:   match  :user1:
func segmentMatch(key string, segments []string) bool {
	fname := HEADER_SEGMENT_SEPARATOR + getFNameFromKey(key) + HEADER_SEGMENT_SEPARATOR
	for _, seg := range segments {
		if !strings.Contains(fname, seg) {
			return false
		}
	}
	return true
}

// getFNameFromKey - return the last portion of the key.
func getFNameFromKey(key string) string {
	if !strings.Contains(key, "/") {
		return key
	}
	sections := strings.Split(key, "/")
	return sections[len(sections)-1]
}

func validateBucketName(bname string) bool {
	if strings.ToLower(bname) != bname {
		return false
	}
	if strings.TrimSpace(bname) != bname {
		return false
	}
	if strings.Contains(bname, " ") {
		return false
	}
	if strings.Contains(bname, ",") {
		return false
	}
	if strings.Contains(bname, "/") {
		return false
	}
	if bname == "status" {
		return false
	}
	return true

}

func isAlias(item *badger.Item) bool {
	if item == nil {
		return false
	}
	return item.UserMeta()&BADGER_FLAG_ALIAS == BADGER_FLAG_ALIAS
}

func SendError(writer http.ResponseWriter, message string, status int) {
	writer.Header().Set(RESP_HEADER_ERROR_MSG, message)
	http.Error(writer, message, status)
}
