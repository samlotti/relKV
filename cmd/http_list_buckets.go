package cmd

import (
	"encoding/json"
	"net/http"
	. "relKV/common"
)

func (b *BucketsDb) listBuckets(writer http.ResponseWriter, request *http.Request) {
	var buckets []*BucketData

	for name := range b.DbBucket {
		bk := &BucketData{
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
		SendError(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("content-type", "application/json")
	writer.Write(data)
}
