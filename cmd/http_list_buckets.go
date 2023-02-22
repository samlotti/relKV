package cmd

import (
	"encoding/json"
	"net/http"
)

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
