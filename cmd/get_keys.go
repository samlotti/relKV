package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	. "relKV/common"
	"strings"
)

// getKeys - the post data should contain a list of keys, will return the keys and values
func (b *BucketsDb) getKeys(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]
	b64 := getHeaderKeyBool("b64", request)
	var db *badger.DB

	//fmt.Printf("bucket:%s\n", bucket)
	bdb, err := b.getDB(bucket)
	if err != nil {
		SendError(writer, err.Error(), http.StatusBadRequest)
		return
	}
	db = bdb

	request.Body = http.MaxBytesReader(writer, request.Body, b.baseTableSize)

	writer.Header().Set("content-type", "application/json")
	writer.Header().Set(RESP_HEADER_RELDB_FUNCTION, "getKeys")
	writer.Write([]byte("[\n"))

	err = db.View(func(txn *badger.Txn) error {

		body, err := io.ReadAll(request.Body)
		if err != nil {
			return err
		}

		keys := strings.Split(string(body), "\n")

		needComma := false
		for _, key := range keys {

			if len(key) == 0 {
				continue
			}

			kv := &KV{
				Key:   key,
				Value: "",
				Error: "",
			}

			item, err := txn.Get([]byte(key))
			if err == nil {

				if isAlias(item) {
					err = item.Value(func(val []byte) error {
						aliasParent, err := txn.Get(val)
						if err != nil {
							return err
						}

						err = aliasParent.Value(func(val []byte) error {
							if b64 {
								kv.Value = base64.StdEncoding.EncodeToString(val)
							} else {
								kv.Value = string(val)
							}
							return nil
						})
						return err
					})

				} else {
					err = item.Value(func(val []byte) error {

						if b64 {
							kv.Value = base64.StdEncoding.EncodeToString(val)
						} else {
							kv.Value = string(val)
						}

						return nil
					})
				}
			}

			if err != nil {
				kv.Error = err.Error()
				err = nil
			}

			if needComma {
				writer.Write([]byte(",\n"))
			}

			data, err := json.Marshal(kv)
			if err != nil {
				fmt.Printf("Err:%s\n", err.Error())
				return err
			}
			writer.Write([]byte("  "))
			writer.Write(data)
			needComma = true

		}
		return err

	})

	if err != nil {
		fmt.Printf("Err:%s\n", err.Error())
		SendError(writer, err.Error(), http.StatusInternalServerError)
	}
	writer.Write([]byte("\n]\n"))
}
