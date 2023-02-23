package cmd

import (
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"sync/atomic"
)

func (b *BucketsDb) delKey(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	rec_deleted := 0

	var db *badger.DB

	var key []byte

	bdb, err := b.getDB(bucket)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	db = bdb

	key = append(key, getKeyByte(request)...)
	keyS := string(key)

	if len(keyS) == 0 {
		http.Error(writer, "key is required", http.StatusBadRequest)
		return
	}

	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		if err != nil {
			return err
		}
		rec_deleted++

		// Do the aliases
		aliasesVal := request.Header.Get(HEADER_ALIAS_KEY)
		if len(aliasesVal) > 0 {
			aliases := strings.Split(aliasesVal, HEADER_ALIAS_SEPARATOR)

			for _, alias := range aliases {
				err = txn.Delete([]byte(alias))
				if err != nil {
					rec_deleted = 0
					txn.Discard()
					return err
				}
				rec_deleted++
			}
		}

		return err
	})

	writer.Header().Set("rec_deleted", fmt.Sprintf("%d", rec_deleted))

	if err != nil {
		if err == badger.ErrKeyNotFound {
			http.Error(writer, badger.ErrKeyNotFound.Error(), http.StatusNotFound)
		} else {
			atomic.AddInt64(&stats.bucketStats[BucketName(bucket)].numError, 1)
			stats.bucketStats[BucketName(bucket)].lastEMessage = err.Error()
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	} else {
		atomic.AddInt64(&stats.bucketStats[BucketName(bucket)].numDelete, 1)
		writer.WriteHeader(http.StatusOK)
	}
}
