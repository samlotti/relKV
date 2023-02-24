package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type BackupData struct {
	Status      string
	LastStart   time.Time
	LastEnd     time.Time
	LastMessage string
}

type BucketStats struct {
	numError      int64
	seqWriteError int64 // How many in a row, resets on an ok write
	numWrites     int64
	numDelete     int64
	lastEMessage  string // the last error message
}

type Stats struct {
	serverStart time.Time
	Backups     map[BucketName]*BackupData
	bucketStats map[BucketName]*BucketStats
}

var StatsInstance = &Stats{}

func (s *Stats) init() {
	s.serverStart = time.Now()
	s.Backups = make(map[BucketName]*BackupData)
	s.bucketStats = make(map[BucketName]*BucketStats)

	for _, bucket := range BucketsInstance.buckets {

		s.addBucket(bucket)

	}
}

func (s *Stats) addBucket(bucket BucketName) {

	s.Backups[bucket] = &BackupData{
		Status:      "",
		LastStart:   s.serverStart,
		LastEnd:     s.serverStart,
		LastMessage: "",
	}

	s.bucketStats[bucket] = &BucketStats{
		numError:      0,
		seqWriteError: 0,
		numWrites:     0,
		lastEMessage:  "",
		numDelete:     0,
	}
}

func (b *BucketsDb) status(writer http.ResponseWriter, request *http.Request) {
	hasErrors := false
	w := bytes.Buffer{}
	w.Write([]byte("<html><body style='background: darkgray'><pre>"))
	w.Write([]byte(fmt.Sprintf("Start: %s\n", StatsInstance.serverStart.Format(time.RFC822))))
	dur := time.Now().Sub(StatsInstance.serverStart)
	w.Write([]byte(fmt.Sprintf("Uptime: %s\n", dur.String())))
	w.Write([]byte(fmt.Sprintf("Current time: %s\n\n", time.Now().Format(time.RFC822))))

	if EnvironmentInstance.GetBoolEnv("NOBACKUP") {
		w.Write([]byte("backupsInstance\n"))
		w.Write([]byte(fmt.Sprintf("** backupsInstance are not enabled\n")))
	} else {
		w.Write([]byte(fmt.Sprintf("backupsInstance - Running at hours: %s\n\n", EnvironmentInstance.GetEnv("BK_HOURS", "?"))))

		w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %s\n", "name", "status", "duration", "lastRun", "last message")))
		for bucket, bstat := range StatsInstance.Backups {
			dur := time.Now().Sub(bstat.LastStart)
			if dur > 24*time.Hour {
				hasErrors = true
				w.Write([]byte(fmt.Sprintf("%-25s: error: backup has not been run\n", bucket)))
			}
			dur = bstat.LastEnd.Sub(bstat.LastStart)
			smsg := bstat.Status
			if bstat.LastStart == StatsInstance.serverStart {
				smsg = "Not run"
			}
			w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %s\n", bucket, smsg, dur.String(), bstat.LastStart.Format(time.RFC822), bstat.LastMessage)))
			if len(bstat.LastMessage) > 0 {
				hasErrors = true
			}
		}
	}

	w.Write([]byte("\n\nWrites\n"))
	w.Write([]byte(fmt.Sprintf("%-20s %15s  %15s  %15s  %15s   %s\n", "name", "#Delete", "#Write", "#WriteErr", "Current Errors", "last error message")))
	for bucket, bstat := range StatsInstance.bucketStats {

		numDelete := atomic.LoadInt64(&bstat.numDelete)
		numError := atomic.LoadInt64(&bstat.numError)
		seqWriteError := atomic.LoadInt64(&bstat.seqWriteError)
		numWrites := atomic.LoadInt64(&bstat.numWrites)

		w.Write([]byte(fmt.Sprintf("%-20s %15d  %15d  %15d  %15d   %s\n", bucket, numDelete, numWrites, numError, seqWriteError, bstat.lastEMessage)))
		if seqWriteError > 10 {
			hasErrors = true
		}
	}

	w.Write([]byte(fmt.Sprintf("\n")))
	if hasErrors {
		w.Write([]byte(fmt.Sprintf("there were some errors listed above\n")))
	}

	w.Write([]byte("</pre></body></html>"))
	if !hasErrors {
		writer.WriteHeader(http.StatusOK)
	} else {
		writer.WriteHeader(http.StatusInternalServerError)
	}
	writer.Write(w.Bytes())

}
