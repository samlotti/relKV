package cmd

import (
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/samlotti/relKV/common"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var reservedKeys = map[string]bool{
	"metrics": true,
	"admin":   true,
	"api":     true,
}

type ServerState int

const Starting ServerState = 0
const Running ServerState = 1
const Stopped ServerState = 2

type BucketsDb struct {
	listenAddrPort string
	DbBucket       map[common.BucketName]*badger.DB
	dbPath         string
	allowCreate    bool
	baseTableSize  int64
	buckets        []common.BucketName
	ServerState    ServerState
	stopChan       chan os.Signal
	authsecret     *AuthSecret
	logfile        string
	logger         *BadgerLogger
	version        string

	Jobs []*common.ScpJob
}

func (b *BucketsDb) shutDownServer() {
	b.stopChan <- syscall.SIGINT
	for {
		time.Sleep(100 * time.Millisecond)
		if b.ServerState == Stopped {
			return
		}
	}
}

func (b *BucketsDb) WaitTillStopped() {
	for {
		time.Sleep(100 * time.Millisecond)
		if b.ServerState == Stopped {
			return
		}
	}
}

func (b *BucketsDb) Init() {

	if len(b.dbPath) == 0 {
		panic("DB_PATH empty or not specified")
	}

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

	b.logger = DefaultLogger(convertLogLevel(EnvironmentInstance.GetEnv("LOG_LEVEL", "INFO")))
}

func (b *BucketsDb) openDBBuckets() {

	// Read the data directory and look for BucketsInstance
	dirs, err := os.ReadDir(b.dbPath)
	if err != nil {
		panic(err)
	}
	for _, entry := range dirs {
		if entry.IsDir() {
			b.addBucket(common.BucketName(entry.Name()))
		}
	}

	b.DbBucket = make(map[common.BucketName]*badger.DB)
	for _, bname := range b.buckets {
		err := b.Open(common.BucketName(bname))
		if err != nil {
			fmt.Printf("error opening bucket:%s", err)
			panic(err)
		}
	}
}

func (b *BucketsDb) Open(name common.BucketName) error {
	dbPath := filepath.Join(b.dbPath, string(name))
	dbOpts := badger.DefaultOptions(dbPath)
	dbOpts = dbOpts.WithLogger(b.logger)
	// dbOpts = dbOpts.WithValueLogFileSize(128 << 20) // 128MB
	// dbOpts = dbOpts.WithIndexCacheSize(128 << 20)   // 128MB
	// dbOpts = dbOpts.WithBaseTableSize(b.baseTableSize)
	dbOpts = dbOpts.WithCompactL0OnClose(true)
	dbOpts = dbOpts.WithBlockCacheSize(256 << 21)

	// Reduce size of bloom filter % false positives
	bloomFalsePositive := EnvironmentInstance.GetBloomFalsePercentage()
	dbOpts = dbOpts.WithBloomFalsePositive(bloomFalsePositive)

	db, err := badger.Open(dbOpts)
	if err != nil {
		return err
	}

	b.DbBucket[name] = db

	return nil
}

func (b *BucketsDb) getDB(bucket string) (*badger.DB, error) {
	if db, ok := b.DbBucket[common.BucketName(bucket)]; ok {
		return db, nil
	}
	return nil, errors.New("bucket not found")
}

func (b *BucketsDb) Close() {
	for _, db := range b.DbBucket {
		db.Close()
	}
}

func (b *BucketsDb) runGC() {
	for {
		time.Sleep(10 * time.Minute)
		if b.ServerState == Stopped {
			return
		}

		//b.logger.Warningf("Running gc loop")

		for name, db := range BucketsInstance.DbBucket {
			//b.logger.Warningf("Name: %s", name)
			if err := db.RunValueLogGC(0.5); err != nil {
				if err != badger.ErrNoRewrite {
					log.Printf("error running gc on:%s", name)
					log.Fatal(err)
				} else {
					atomic.AddInt64(&StatsInstance.bucketStats[common.BucketName(name)].numGCNR, 1)
				}
			} else {
				atomic.AddInt64(&StatsInstance.bucketStats[common.BucketName(name)].numGC, 1)
			}
		}
	}
}

// getListenAddr - returns as http://host:port
func (b *BucketsDb) getListenAddr() string {
	hostport := b.listenAddrPort
	if strings.HasPrefix(b.listenAddrPort, ":") {
		hostport = "localhost" + hostport
	}
	return "http://" + hostport
}

func (b *BucketsDb) WaitTillStarted() {

	for {
		time.Sleep(100 * time.Millisecond)

		_, err := http.Get(b.getListenAddr() + "/status")
		if err != nil {
			fmt.Printf("err: %s\n", err)
			continue
		}

		if b.ServerState == Running {
			return
		}
		if b.ServerState == Stopped {
			panic("server stopped while waiting for it sto start")
		}
	}
}

func (b *BucketsDb) addBucket(name common.BucketName) {
	if !validateBucketName(string(name)) {
		panic(fmt.Sprintf("bad bucket name %s", name))
	}
	for _, e := range b.buckets {
		if e == name {
			return
		}
	}
	b.buckets = append(b.buckets, common.BucketName(name))
	StatsInstance.addBucket(name)
}
