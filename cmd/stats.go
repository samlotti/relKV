package cmd

import (
	"bytes"
	"fmt"
	"github.com/samlotti/relKV/common"
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
	numGC         int64
	numGCNR       int64
	lastEMessage  string // the last error message
}

type Stats struct {
	serverStart   time.Time
	Backups       map[common.BucketName]*BackupData
	bucketStats   map[common.BucketName]*BucketStats
	LastBKRunLoop time.Time
	LastBKStart   time.Time
}

var StatsInstance = &Stats{}

func (s *Stats) init() {
	s.serverStart = time.Now()
	s.Backups = make(map[common.BucketName]*BackupData)
	s.bucketStats = make(map[common.BucketName]*BucketStats)

	for _, bucket := range BucketsInstance.buckets {
		s.addBucket(bucket)
	}
}

func (s *Stats) addBucket(bucket common.BucketName) {

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
	w.Write([]byte(fmt.Sprintf("relKv %s\n", BucketsInstance.version)))
	w.Write([]byte(fmt.Sprintf("Start: %s\n", StatsInstance.serverStart.Format(time.RFC822))))
	dur := time.Now().Sub(StatsInstance.serverStart)
	w.Write([]byte(fmt.Sprintf("Uptime: %s\n", dur.String())))
	w.Write([]byte(fmt.Sprintf("Current time: %s\n\n", time.Now().Format(time.RFC822))))
	w.Write([]byte("===================================\n\n"))

	var hourGrace = time.Duration(EnvironmentInstance.GetBackupGraceHours()) * time.Hour

	if EnvironmentInstance.GetBoolEnv("NOBACKUP") {
		w.Write([]byte("backupsInstance\n"))
		w.Write([]byte(fmt.Sprintf("** backupsInstance are not enabled\n")))
	} else {
		w.Write([]byte(fmt.Sprintf("backupsInstance - Running at hours: %s\n", EnvironmentInstance.GetEnv("BK_HOURS", "?"))))
		w.Write([]byte(fmt.Sprintf("Age for backup before its considered failed: %s\n", hourGrace)))

		w.Write([]byte(fmt.Sprintf("last check loop -  %s\n", StatsInstance.LastBKRunLoop.Format(time.RFC822))))
		w.Write([]byte(fmt.Sprintf("last start      -  %s\n\n", StatsInstance.LastBKStart.Format(time.RFC822))))

		w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %s\n", "name", "status", "duration", "lastRun", "last message")))
		keys := sortBucketKeys(StatsInstance.bucketStats)
		for _, bucket := range keys {
			bstat := StatsInstance.Backups[bucket]

			dur := time.Now().Sub(bstat.LastStart)
			if dur > hourGrace {
				hasErrors = true
				w.Write([]byte(fmt.Sprintf("%-25s: error: backup has not been run\n", bucket)))
			}
			dur = bstat.LastEnd.Sub(bstat.LastStart)
			if bstat.Status == "running" {
				dur = time.Now().Sub(bstat.LastStart)
			}
			smsg := bstat.Status
			if bstat.LastStart == StatsInstance.serverStart {
				smsg = "Not run"
			}
			w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %s\n", bucket, smsg, dur.String(), bstat.LastStart.Format(time.RFC822), bstat.LastMessage)))
			//if len(bstat.LastMessage) > 0 {
			//	hasErrors = true
			//}
			// zipping file is a valid message
			if len(bstat.LastMessage) > 0 &&
				bstat.LastMessage != "Creating backup" &&
				bstat.LastMessage != "Zipping file" {
				hasErrors = true
			}
		}
		//
		//w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %s\n", "name", "status", "duration", "lastRun", "last message")))
		//keys = sortBucketKeys(StatsInstance.bucketStats)
		//for _, bucket := range keys {
		//	bstat := StatsInstance.Scps[bucket]
		//
		//	dur := time.Now().Sub(bstat.LastStart)
		//	if dur > 24*time.Hour {
		//		hasErrors = true
		//		w.Write([]byte(fmt.Sprintf("%-25s: error: backup has not been run\n", bucket)))
		//	}
		//	dur = bstat.LastEnd.Sub(bstat.LastStart)
		//	smsg := bstat.Status
		//	if bstat.LastStart == StatsInstance.serverStart {
		//		smsg = "Not run"
		//	}
		//	w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %s\n", bucket, smsg, dur.String(), bstat.LastStart.Format(time.RFC822), bstat.LastMessage)))
		//	if len(bstat.LastMessage) > 0 {
		//		hasErrors = true
		//	}
		//}

		if len(b.Jobs) > 0 {
			w.Write([]byte("\n\n===================================\n"))
			w.Write([]byte("Scp jobs to remote\n"))
			// if common.ScpEnvInstance.IsEnabled() {
			w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %-25s %s\n", "bucket", "status", "duration", "next Send", "last Send", "message")))
			for _, job := range b.Jobs {
				dur = job.LastEnd.Sub(job.LastStart)
				nextSend := job.NextSend.Format(time.RFC822)
				lastSend := job.LastStart.Format(time.RFC822)
				smsg := ""
				switch job.Status {
				case common.ScpError:
					smsg = "Error"
				case common.ScpRunning:
					smsg = "Running"
					dur = time.Now().Sub(job.LastStart)
					lastSend = "running"
				case common.ScpPending:
					smsg = "Pending"
				case common.ScpComplete:
					smsg = "Completed"
					nextSend = ""
				}

				w.Write([]byte(fmt.Sprintf("%-20s %-15s %-25s %-25s %-25s %s\n", job.BucketName, smsg, dur.String(), nextSend, lastSend, job.Message)))

				// zipping file is a valid message
				if len(job.Message) > 0 {
					hasErrors = true
				}

				if time.Now().Sub(job.LastStart) > hourGrace {
					hasErrors = true
					w.Write([]byte(fmt.Sprintf("%-25s: error: backup has not been run\n", job.BucketName)))
				}

			}
		}

	}
	w.Write([]byte("\n\n===================================\n"))

	w.Write([]byte("\nWrites\n"))
	w.Write([]byte(fmt.Sprintf("%-20s %15s  %15s  %15s  %15s   %s\n", "name", "#Delete", "#Write", "#WriteErr", "Current Errors", "last error message")))

	keys := sortBucketKeys(StatsInstance.bucketStats)
	for _, key := range keys {
		bstat := StatsInstance.bucketStats[key]
		numDelete := atomic.LoadInt64(&bstat.numDelete)
		numError := atomic.LoadInt64(&bstat.numError)
		seqWriteError := atomic.LoadInt64(&bstat.seqWriteError)
		numWrites := atomic.LoadInt64(&bstat.numWrites)

		w.Write([]byte(fmt.Sprintf("%-20s %15d  %15d  %15d  %15d   %s\n", key, numDelete, numWrites, numError, seqWriteError, bstat.lastEMessage)))
		if seqWriteError > 10 {
			hasErrors = true
		}
	}

	w.Write([]byte(fmt.Sprintf("\n")))
	if hasErrors {
		w.Write([]byte(fmt.Sprintf("there were some errors listed above\n")))
	}

	w.Write([]byte("\nGarbage Collection Cycles\n"))
	w.Write([]byte(fmt.Sprintf("%-20s %15s %15s\n", "name", "#Cycles", "No rewrite")))

	keys = sortBucketKeys(StatsInstance.bucketStats)
	for _, key := range keys {
		bstat := StatsInstance.bucketStats[key]
		numCycles := atomic.LoadInt64(&bstat.numGC)
		NRnumCycles := atomic.LoadInt64(&bstat.numGCNR)

		w.Write([]byte(fmt.Sprintf("%-20s %15d %15d\n", key, numCycles, NRnumCycles)))
	}

	w.Write([]byte("\nMemory related\n"))
	w.Write([]byte(fmt.Sprintf("BK_NUM_GO=%d  lower = less memory during backup\n", EnvironmentInstance.GetBackupGoRoutineNumber())))
	w.Write([]byte(fmt.Sprintf("BLOOM_FALSE_PERCENTAGE=%f  0=off, less memory as approach to 0.99\n", EnvironmentInstance.GetBloomFalsePercentage())))
	// w.Write([]byte(fmt.Sprintf("BK_ZIP=%t  true will zip the file\n", EnvironmentInstance.GetBoolEnv("BK_ZIP"))))

	w.Write([]byte("</pre></body></html>"))
	if !hasErrors {
		writer.WriteHeader(http.StatusOK)
	} else {
		writer.WriteHeader(http.StatusInternalServerError)
	}
	writer.Write(w.Bytes())

}
