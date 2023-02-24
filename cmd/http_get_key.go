package cmd

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"net/http"
	"relKV/common"
)

func (b *BucketsDb) getKey(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	writer.Header().Set(common.RESP_HEADER_RELDB_FUNCTION, "getKey")

	var db *badger.DB

	var key []byte

	bdb, err := b.getDB(bucket)
	if err != nil {
		SendError(writer, err.Error(), http.StatusBadRequest)
		return
	}
	db = bdb

	key = append(key, getKeyByte(request)...)

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				SendError(writer, err.Error(), http.StatusNotFound)
				err = nil
			}
			return err
		}

		if isAlias(item) {
			err := item.Value(func(val []byte) error {
				aliasParent, err := txn.Get(val)
				if err != nil {
					return err
				}

				err = aliasParent.Value(func(val []byte) error {
					writer.Write(val)
					return nil
				})
				return err
			})
			// ??
			if err != nil {
				SendError(writer, err.Error(), http.StatusNotFound)
				return nil
			}

		} else {
			return item.Value(func(val []byte) error {
				writer.Write(val)
				return nil
			})
		}
		return nil

	})

	if err != nil {
		SendError(writer, err.Error(), http.StatusInternalServerError)
	}
}
