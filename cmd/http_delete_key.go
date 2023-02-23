package cmd

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"net/http"
	"sync/atomic"
)

func (b *BucketsDb) delKey(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	var db *badger.DB

	var key []byte

	bdb, err := b.getDB(bucket)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}
	db = bdb

	key = append(key, getKeyByte(request)...)

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})

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
