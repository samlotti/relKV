package backup

import (
	"bufio"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	. "github.com/samlotti/relKV/cmd"
	"github.com/samlotti/relKV/common"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"time"
)

type Backups struct {
	bkfolder  string
	lastBkHr  int
	lastBkDay int
	hourList  []int
	buckets   *BucketsDb
}

var BackupsInstance *Backups

func BackupsInit(buckets *BucketsDb) {
	BackupsInstance = &Backups{
		buckets: buckets,
	}

	BackupsInstance.lastBkHr = -1
	BackupsInstance.bkfolder = EnvironmentInstance.GetEnv("BK_PATH", "")
	BackupsInstance.hourList = EnvironmentInstance.GetIntArray("BK_HOURS")

	path, err := filepath.Abs(BackupsInstance.bkfolder)
	if err != nil {
		panic(err)
	}
	log.Printf("backup directory:%s", path)
	BackupsInstance.bkfolder = path

	if _, err := os.Stat(BackupsInstance.bkfolder); os.IsNotExist(err) {
		log.Printf("directory not found, %s, please create it first", BackupsInstance.bkfolder)
		panic(err)
	}
}

func (b *Backups) Run() {
	for {
		time.Sleep(15 * time.Second)

		if b.buckets.ServerState == Stopped {
			return
		}

		b.runBk()

	}
}

func (b *Backups) createBackup(name common.BucketName, db *badger.DB) {

	bkgonum := EnvironmentInstance.GetBackupGoRoutineNumber()

	suffixDay := EnvironmentInstance.GetBoolEnv("BK_SUFFIX_DAY")
	suffixHour := EnvironmentInstance.GetBoolEnv("BK_SUFFIX_HOUR")
	bkZip := EnvironmentInstance.GetBoolEnv("BK_ZIP")

	StatsInstance.Backups[name].LastStart = time.Now()
	StatsInstance.Backups[name].Status = "running"
	StatsInstance.Backups[name].LastMessage = "Creating backup"

	// log.Printf("Backup started: %s\n", name)
	origBfname := CreateBackupFilename(name, suffixDay, suffixHour)
	bfname := origBfname
	//if bkZip {
	//	bfname = AddZipToFilename(bfname)
	//}

	destFilename := path.Join(b.bkfolder, bfname)
	f, err := os.Create(destFilename)
	if err != nil {
		StatsInstance.Backups[name].LastEnd = time.Now()
		StatsInstance.Backups[name].Status = "failed"
		StatsInstance.Backups[name].LastMessage = "error creating backup: " + err.Error()
		return
	}

	destFilenameZip := path.Join(b.bkfolder, AddZipToFilename(bfname))
	os.Remove(destFilenameZip)

	var w io.Writer
	//var wz *zip.Writer
	var wb *bufio.Writer

	// Dont zip in the process.t
	//if bkZip {
	//	wz = zip.NewWriter(f)
	//	w, err = wz.Create(origBfname)
	//	if err != nil {
	//		StatsInstance.Backups[name].Status = "failed"
	//		StatsInstance.Backups[name].LastMessage = "error creating zip file: " + err.Error()
	//		return
	//	}
	//} else {
	wb = bufio.NewWriter(f)
	w = wb
	//}

	// _, err = db.Backup(w, 0)

	stream := db.NewStream()
	stream.LogPrefix = fmt.Sprintf("backup.stream: %s", name)
	stream.NumGo = bkgonum // Default is 16 -- reduce memory usage
	_, err = stream.Backup(w, 0)

	StatsInstance.Backups[name].LastEnd = time.Now()

	failed := false
	if err != nil {
		StatsInstance.Backups[name].Status = "failed"
		StatsInstance.Backups[name].LastMessage = "error creating backup: " + err.Error()
		failed = true
		return
	} else {
		if bkZip {
			// StatsInstance.Backups[name].Status = "completed"
			StatsInstance.Backups[name].LastMessage = "Zipping file"
		}
	}
	//if wz != nil {
	//	wz.Flush()
	//	wz.Close()
	//} else {
	wb.Flush()
	//}
	f.Close()

	if failed {
		return
	}

	// Do the zip outside the application.
	if bkZip {
		cmdStruct := exec.Command("gzip", destFilename)
		_, err := cmdStruct.Output()
		if err != nil {
			// fmt.Println(err)
			StatsInstance.Backups[name].Status = "gzip failed."
			StatsInstance.Backups[name].LastMessage = "error zipping: " + err.Error() + " > " + destFilenameZip
			failed = true
		} else {
			StatsInstance.Backups[name].Status = "completed"
			StatsInstance.Backups[name].LastMessage = ""
		}
	}

	if !failed {
		if bkZip {
			go ScpEnvInstance.AddScpJob(name, destFilenameZip)
		} else {
			go ScpEnvInstance.AddScpJob(name, destFilename)
		}
	}

}

func (b *Backups) runBk() {
	// Don't let it die, try again next time
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Println("error in backup", rec)
			fmt.Printf("%s", debug.Stack())
		}
	}()

	StatsInstance.LastBKRunLoop = time.Now()

	runBk := false

	//bug: if one hour in list it will only run once.
	// need to add day
	if time.Now().Hour() != b.lastBkHr || time.Now().Day() != b.lastBkDay {
		for _, h := range b.hourList {
			if h == time.Now().Hour() && time.Now().Day() != b.lastBkDay {
				runBk = true
				break
			}
		}
	}
	if runBk {
		b.lastBkHr = time.Now().Hour()
		b.lastBkDay = time.Now().Day()
		StatsInstance.LastBKStart = time.Now()

		for name, db := range BucketsInstance.DbBucket {
			b.createBackup(name, db)
		}
	}
}
