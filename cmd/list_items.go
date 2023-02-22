package cmd

import (
	"encoding/base64"
	"encoding/json"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"math"
	"net/http"
)

// listKeys - returns keys in the bucket and optionally the contents
// parameters supported:
//   skip, max <- paging support
//   rel <- relationships in filename portion
//   prefix <- limit to prefixes
//   values <- t/f  default is false
func (b *BucketsDb) listKeys(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	skip := getHeaderKeyInt("skip", 0, request)
	max := getHeaderKeyInt("max", math.MaxInt, request)
	getValues := getHeaderKeyBool("values", request)
	b64 := getHeaderKeyBool("b64", request)
	segments := getSegments(getHeaderKey("segments", request))

	var db *badger.DB

	bdb, err := b.getDB(bucket)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}
	db = bdb

	writer.Header().Set("content-type", "application/json")
	writer.Write([]byte("[\n"))

	rnum := 0
	count := 0
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		opts.PrefetchValues = getValues

		it := txn.NewIterator(opts)
		defer it.Close()

		var prefix []byte

		userPrefix := getHeaderKey("prefix", request)
		if userPrefix != "" {
			prefix = append(prefix, []byte(userPrefix)...)
		}

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			keyStr := string(key)
			if err != nil {
				return err
			}
			kv := &KV{
				Key:   keyStr,
				Value: "",
				Error: "",
			}

			// Additional selection
			if segments != nil {
				if !segmentMatch(keyStr, segments) {
					continue
				}
			}

			rnum += 1
			if rnum <= skip {
				continue
			}

			count += 1
			if count > max {
				return nil
			}

			if getValues {
				err := item.Value(func(val []byte) error {
					if b64 {
						kv.Value = base64.StdEncoding.EncodeToString(val)
					} else {
						kv.Value = string(val)
					}
					return nil
				})
				if err != nil {
					return err
				}
			}

			data, err := json.Marshal(kv)
			if err != nil {
				return err
			}
			writer.Write(data)
			writer.Write([]byte("\n"))

		}
		return nil
	})

	// Send any error
	if err != nil {
		// little late for setting the status code
		kv := &KV{
			Key:   "",
			Value: "",
			Error: err.Error(),
		}

		data, err := json.Marshal(kv)
		if err != nil {
			writer.Write(data)
		}
	}

	writer.Write([]byte("]\n"))
}
