package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var reservedKeys = map[string]bool{
	"metrics": true,
	"admin":   true,
	"api":     true,
}

type BucketsDb struct {
	dbBucket      map[BucketName]*badger.DB
	dbPath        string
	baseTableSize int64
	buckets       []BucketName
}

func (b *BucketsDb) Init() {
	path, err := filepath.Abs(b.dbPath)
	if err != nil {
		panic(err)
	}
	log.Printf("data directory:%s", path)
	b.dbPath = path

	if _, err := os.Stat(b.dbPath); os.IsNotExist(err) {
		log.Printf("directory not found, %s, please create it first", b.dbPath)
		panic(err)
	}
}

func (b *BucketsDb) openDBBuckets() {
	b.dbBucket = make(map[BucketName]*badger.DB)
	for _, bname := range b.buckets {
		err := b.Open(BucketName(bname))
		if err != nil {
			fmt.Printf("error opening bucket:%s", err)
			panic(err)
		}
	}
}

func (b *BucketsDb) Open(name BucketName) error {
	dbPath := filepath.Join(b.dbPath, string(name))
	dbOpts := badger.DefaultOptions(dbPath)
	dbOpts = dbOpts.WithLogger(DefaultLogger(INFO))
	dbOpts = dbOpts.WithValueLogFileSize(128 << 20) // 128MB
	dbOpts = dbOpts.WithIndexCacheSize(128 << 20)   // 128MB
	dbOpts = dbOpts.WithBaseTableSize(b.baseTableSize)
	dbOpts = dbOpts.WithCompactL0OnClose(true)

	db, err := badger.Open(dbOpts)
	if err != nil {
		return err
	}

	b.dbBucket[name] = db

	return nil
}

func (b *BucketsDb) getDB(bucket string) (*badger.DB, error) {
	if db, ok := b.dbBucket[BucketName(bucket)]; ok {
		return db, nil
	}
	return nil, errors.New("bucket not found")
}

func (b *BucketsDb) Close() {
	for _, db := range b.dbBucket {
		db.Close()
	}
}

func (b *BucketsDb) runGC() {
	for {
		time.Sleep(10 * time.Minute)
		for name, db := range b.dbBucket {
			if err := db.RunValueLogGC(0.7); err != nil {
				if err != badger.ErrNoRewrite {
					log.Printf("error running gc on:%s", name)
					log.Fatal(err)
				}
			}
		}
	}
}

type bucketData struct {
	Name     string `json:"name"`
	Error    string `json:"error,omitempty"`
	LsmSize  int64  `json:"lsmSize"`
	VlogSize int64  `json:"VlogSize"`
}

func (b *BucketsDb) listBuckets(writer http.ResponseWriter, request *http.Request) {
	var buckets []*bucketData

	for name := range b.dbBucket {
		bk := &bucketData{
			Name: string(name),
		}
		buckets = append(buckets, bk)

		db, err := b.getDB(string(name))
		if err != nil {
			bk.Error = err.Error()
		} else {
			lsm, vlog := db.Size()
			bk.LsmSize = lsm
			bk.VlogSize = vlog
		}
	}

	data, err := json.Marshal(buckets)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("content-type", "application/json")
	writer.Write(data)
}

type KV struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

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
