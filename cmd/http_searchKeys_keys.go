package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"math"
	"net/http"
)

// searchKeys - returns keys in the bucket and optionally the contents
// parameters supported:
//   skip, max <- paging support
//   rel <- relationships in filename portion
//   prefix <- limit to prefixes
//   values <- t/f  default is false
func (b *BucketsDb) searchKeys(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	skip := getHeaderKeyInt(HEADER_SKIP_KEY, 0, request)
	max := getHeaderKeyInt(HEADER_MAX_KEY, math.MaxInt, request)
	getValues := getHeaderKeyBool(HEADER_VALUES_KEY, request)
	b64 := getHeaderKeyBool(HEADER_B64_KEY, request)
	segments := getSegments(getHeaderKey("segments", request))
	explain := getHeaderKeyInt(HEADER_EXPLAIN_KEY, 0, request) == 1

	ex_rows_read := 0
	ex_rows_selected := 0
	ex_rows_skipped := 0

	var db *badger.DB

	bdb, err := b.getDB(bucket)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}
	db = bdb

	writer.Header().Set("content-type", "application/json")
	writer.Header().Set(RESP_HEADER_KVDB_FUNCTION, "searchKeys")

	if !explain {
		writer.Write([]byte("[\n"))
	}

	rnum := 0
	count := 0
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		opts.PrefetchValues = getValues

		it := txn.NewIterator(opts)
		defer it.Close()

		var prefix []byte

		userPrefix := getHeaderKey(HEADER_PREFIX_KEY, request)
		if userPrefix != "" {
			prefix = append(prefix, []byte(userPrefix)...)
		}

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			ex_rows_read++

			item := it.Item()
			key := item.Key()
			keyStr := string(key)

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

			// Resolve to the real value if alias!
			// If not found ignore the alias entry
			if getValues {
				if item.UserMeta()&BADGER_FLAG_ALIAS == BADGER_FLAG_ALIAS {
					err := item.Value(func(val []byte) error {
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
					// ??
					if err != nil {
						continue
					}

				}
			}

			rnum += 1
			if rnum <= skip {
				ex_rows_skipped++
				continue
			}

			count += 1
			if count > max {
				return nil
			}

			ex_rows_selected++

			if ex_rows_selected > 1 {
				if !explain {
					writer.Write([]byte(",\n"))
				}
			}

			if getValues && len(kv.Value) == 0 {
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

			if !explain {
				writer.Write([]byte("  "))
				writer.Write(data)
			}

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
			if !explain {
				writer.Write(data)
			}
		}
	}
	if !explain {
		writer.Write([]byte("\n"))
		writer.Write([]byte("]\n"))
	} else {
		writer.Header().Set("ex_row_read", fmt.Sprint(ex_rows_read))
		writer.Header().Set("ex_rows_selected", fmt.Sprint(ex_rows_selected))
		writer.Header().Set("ex_rows_skipped", fmt.Sprint(ex_rows_skipped))
		writer.Write([]byte("[]\n"))

	}
}
