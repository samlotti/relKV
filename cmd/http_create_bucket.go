package cmd

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func (b *BucketsDb) createBucket(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	bucket := vars["bucket"]

	// bucket = strings.TrimSpace(strings.ToLower(bucket))

	if !validateBucketName(bucket) {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if b.DbBucket[BucketName(bucket)] != nil {
		writer.WriteHeader(http.StatusCreated)
		return
	}

	err := b.Open(BucketName(bucket))
	if err == nil {
		b.addBucket(BucketName(bucket))
		writer.WriteHeader(http.StatusCreated)
	} else {
		log.Println(fmt.Sprintf("error creating bucket:%s, %s", bucket, err))
		SendError(writer, "error creating bucket", http.StatusInternalServerError)
	}

}
