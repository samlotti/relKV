package backup

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/povsister/scp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	. "kvDb/cmd"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

type Backups struct {
	bkfolder string
	lastBkHr int
	hourList []int
	buckets  *BucketsDb
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
		time.Sleep(1 * time.Minute)

		if b.buckets.ServerState == Stopped {
			return
		}

		b.runBk()

	}
}

func (b *Backups) createBackup(name BucketName, db *badger.DB) {

	suffixDay := EnvironmentInstance.GetBoolEnv("BK_SUFFIX_DAY")
	suffixHour := EnvironmentInstance.GetBoolEnv("BK_SUFFIX_HOUR")
	bkZip := EnvironmentInstance.GetBoolEnv("BK_ZIP")

	StatsInstance.Backups[name].LastStart = time.Now()
	StatsInstance.Backups[name].Status = "running"
	StatsInstance.Backups[name].LastMessage = ""

	// log.Printf("Backup started: %s\n", name)
	origBfname := CreateBackupFilename(name, suffixDay, suffixHour)
	bfname := origBfname
	if bkZip {
		bfname = AddZipToFilename(bfname)
	}

	destFilename := path.Join(b.bkfolder, bfname)
	f, err := os.Create(destFilename)
	if err != nil {
		StatsInstance.Backups[name].LastEnd = time.Now()
		StatsInstance.Backups[name].Status = "failed"
		StatsInstance.Backups[name].LastMessage = "error creating backup: " + err.Error()
		return
	}

	var w io.Writer
	var wz *zip.Writer
	var wb *bufio.Writer

	if bkZip {
		wz = zip.NewWriter(f)
		w, err = wz.Create(origBfname)
		if err != nil {
			StatsInstance.Backups[name].Status = "failed"
			StatsInstance.Backups[name].LastMessage = "error creating zip file: " + err.Error()
			return
		}
	} else {
		wb = bufio.NewWriter(f)
		w = wb
	}

	_, err = db.Backup(w, 0)
	StatsInstance.Backups[name].LastEnd = time.Now()
	if err != nil {
		StatsInstance.Backups[name].Status = "failed"
		StatsInstance.Backups[name].LastMessage = "error creating backup: " + err.Error()
		return
	} else {
		StatsInstance.Backups[name].Status = "completed"
		StatsInstance.Backups[name].LastMessage = ""
	}
	if wz != nil {
		wz.Flush()
		wz.Close()
	} else {
		wb.Flush()
	}
	f.Close()

	b.sendSCP(name, destFilename)

}

func (b *Backups) runBk() {
	// Don't let it die, try again next time
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Println("error in backup", rec)
			fmt.Printf("%s", debug.Stack())
		}
	}()

	runBk := false
	if time.Now().Hour() != b.lastBkHr {
		for _, h := range b.hourList {
			if h == time.Now().Hour() {
				runBk = true
				break
			}
		}
	}
	if runBk {
		b.lastBkHr = time.Now().Hour()

		for name, db := range BucketsInstance.DbBucket {
			b.createBackup(name, db)
		}
	}
}

func (b *Backups) sendSCP(name BucketName, fileToSend string) {
	scpHost := EnvironmentInstance.GetEnv("BK_SCP_HOST", "")
	scpDir := EnvironmentInstance.GetEnv("BK_SCP_DIR", "")
	scpUname := EnvironmentInstance.GetEnv("BK_SCP_UNAME", "")
	scpUpwd := EnvironmentInstance.GetEnv("BK_SCP_UPWD", "")
	scpKeypath := EnvironmentInstance.GetEnv("BK_SCP_PATH_TO_KEY", "")
	suffixDay := EnvironmentInstance.GetBoolEnv("BK_SCP_SUFFIX_DAY")
	suffixHour := EnvironmentInstance.GetBoolEnv("BK_SCP_SUFFIX_HOUR")
	bkZip := EnvironmentInstance.GetBoolEnv("BK_ZIP")

	if len(scpHost) == 0 ||
		len(scpDir) == 0 ||
		len(scpUname) == 0 ||
		(len(scpUpwd) == 0 &&
			len(scpKeypath) == 0) {
		return
	}

	StatsInstance.Backups[name].Status = "scp"

	scpDestName := CreateBackupFilename(name, suffixDay, suffixHour)
	if bkZip {
		scpDestName = AddZipToFilename(scpDestName)
	}

	var sshConf *ssh.ClientConfig

	if len(scpUpwd) > 0 {
		log.Printf("scp using name/password %s. %s", scpUname, strings.Repeat("x", len(scpUpwd)))
		sshConf = scp.NewSSHConfigFromPassword(scpUname, scpUpwd)
	} else {
		log.Printf("scp using name/private key")
		privPEM, err := ioutil.ReadFile(scpKeypath)
		if err != nil {
			StatsInstance.Backups[name].LastMessage = fmt.Sprintf("error creating scp config read private key %s", err.Error())
			StatsInstance.Backups[name].Status = "error"
			return
		}
		sshConf, err = scp.NewSSHConfigFromPrivateKey(scpUname, privPEM)
		if err != nil {
			StatsInstance.Backups[name].LastMessage = fmt.Sprintf("error creating scp config with private key %s", err.Error())
			StatsInstance.Backups[name].Status = "error"
			return
		}

	}
	scpClient, err := scp.NewClient(scpHost, sshConf, &scp.ClientOption{})
	if err != nil {
		StatsInstance.Backups[name].LastMessage = fmt.Sprintf("error creating scp client %s", err.Error())
		StatsInstance.Backups[name].Status = "error"
		return
	}
	defer scpClient.Close()

	transferOptions := &scp.FileTransferOption{
		Context:      context.Background(),
		Timeout:      30 * time.Second,
		PreserveProp: true,
	}
	destFile := path.Join(scpDir, scpDestName)
	log.Printf("Scp %s:%s -> %s", scpHost, fileToSend, destFile)
	err = scpClient.CopyFileToRemote(fileToSend, destFile, transferOptions)
	if err != nil {
		log.Printf("error sending file:%s, %s", name, err)
		StatsInstance.Backups[name].LastMessage = fmt.Sprintf("error during send %s", err.Error())
		StatsInstance.Backups[name].Status = "error"
		return
	}

	StatsInstance.Backups[name].LastEnd = time.Now()
	StatsInstance.Backups[name].Status = "completed"
}
