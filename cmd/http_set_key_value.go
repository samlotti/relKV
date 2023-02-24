package cmd

import (
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	. "relKV/common"
	"strings"
	"sync/atomic"
)

func (b *BucketsDb) setKey(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	aliasesVal := request.Header.Get(HEADER_ALIAS_KEY)
	aliases := strings.Split(aliasesVal, HEADER_ALIAS_SEPARATOR)

	var db *badger.DB

	key := getKeyByte(request)
	keyS := string(key)

	if len(keyS) == 0 {
		SendError(writer, "key is required", http.StatusBadRequest)
		return
	}

	bdb, err := b.getDB(bucket)
	if err != nil {
		SendError(writer, err.Error(), http.StatusBadRequest)
		return
	}
	db = bdb

	if !isKeyValid(keyS) {
		SendError(writer, "key is has bad characters", http.StatusBadRequest)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, b.baseTableSize)

	status := http.StatusCreated
	dupKey := ""
	// log.Printf("set key: %s", keyS)
	err = db.Update(func(txn *badger.Txn) error {
		bodyBytes, err := io.ReadAll(request.Body)
		if err != nil {
			return err
		}

		existing, err := txn.Get(key)
		if err != nil {
		} else {
			if isAlias(existing) {
				// This is no good
				status = http.StatusBadRequest
				dupKey = string(key)
				err = fmt.Errorf("current key is an aliase, cannot update alias directly")
				return err
			}
		}

		e := badger.NewEntry(key, bodyBytes)
		err = txn.SetEntry(e)

		if err != nil {
			txn.Discard()
			return err
		}

		if len(aliasesVal) > 0 {
			for _, alias := range aliases {

				item, err := txn.Get([]byte(alias))
				if err == nil {
					if isAlias(item) {
						currentAliasValue := ""
						err = item.Value(func(val []byte) error {
							currentAliasValue = string(val)
							return nil
						})
						if err != nil {
							txn.Discard()
							return err
						}
						if currentAliasValue != string(key) {
							status = http.StatusBadRequest
							dupKey = alias
							txn.Discard()
							return fmt.Errorf("alias duplicate key")
						}
					} else {
						// Not an alias
						status = http.StatusBadRequest
						dupKey = alias
						txn.Discard()
						return fmt.Errorf("alias tried to overrite regular key")
					}
				}

				e := badger.NewEntry([]byte(alias), key).WithMeta(BADGER_FLAG_ALIAS)
				err = txn.SetEntry(e)
				if err != nil {
					txn.Discard()
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		b.logger.Debugf("error:%s", err)

		atomic.AddInt64(&StatsInstance.bucketStats[BucketName(bucket)].numError, 1)
		atomic.AddInt64(&StatsInstance.bucketStats[BucketName(bucket)].seqWriteError, 1)
		StatsInstance.bucketStats[BucketName(bucket)].lastEMessage = err.Error()
		if len(dupKey) > 0 {
			writer.Header().Set(RESP_HEADER_DUPLICATE_ERROR, dupKey)
		}
		if status == http.StatusCreated {
			SendError(writer, err.Error(), http.StatusInternalServerError)
		} else {
			SendError(writer, err.Error(), status)
		}

	} else {
		atomic.AddInt64(&StatsInstance.bucketStats[BucketName(bucket)].numWrites, 1)
		atomic.StoreInt64(&StatsInstance.bucketStats[BucketName(bucket)].seqWriteError, 0)
		writer.WriteHeader(status)
	}

}
