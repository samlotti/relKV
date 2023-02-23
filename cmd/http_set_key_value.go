package cmd

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"io"
	"net/http"
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
		http.Error(writer, "key is required", http.StatusBadRequest)
		return
	}

	bdb, err := b.getDB(bucket)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	db = bdb

	if !isKeyValid(keyS) {
		http.Error(writer, "key is has bad characters", http.StatusBadRequest)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, b.baseTableSize)

	// log.Printf("set key: %s", keyS)
	err = db.Update(func(txn *badger.Txn) error {
		b, err := io.ReadAll(request.Body)
		if err != nil {
			return err
		}
		e := badger.NewEntry(key, b)
		err = txn.SetEntry(e)

		if err != nil {
			txn.Discard()
			return err
		}

		if len(aliasesVal) > 0 {
			for _, alias := range aliases {
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
		atomic.AddInt64(&stats.bucketStats[BucketName(bucket)].numError, 1)
		atomic.AddInt64(&stats.bucketStats[BucketName(bucket)].seqWriteError, 1)
		stats.bucketStats[BucketName(bucket)].lastEMessage = err.Error()
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {
		atomic.AddInt64(&stats.bucketStats[BucketName(bucket)].numWrites, 1)
		atomic.StoreInt64(&stats.bucketStats[BucketName(bucket)].seqWriteError, 0)

	}
	writer.WriteHeader(http.StatusCreated)
}
