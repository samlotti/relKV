package cmd

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"net/http"
)

func (b *BucketsDb) getKey(writer http.ResponseWriter, request *http.Request) {
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

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			writer.Write(val)
			return nil
		})
	})

	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
