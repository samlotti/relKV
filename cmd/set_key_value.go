package cmd

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"sync/atomic"
)

func (b *BucketsDb) setKey(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	var db *badger.DB

	key := getKeyByte(request)
	keyS := string(key)

	if len(keyS) == 0 {
		http.Error(writer, "key is required", http.StatusBadRequest)
		return
	}

	bdb, err := b.getDB(bucket)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
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
		return txn.SetEntry(e)
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
