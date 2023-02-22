package cmd

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

var backupsInstance *Backups

func BackupsInit(buckets *BucketsDb) {
	backupsInstance = &Backups{
		buckets: buckets,
	}

	backupsInstance.lastBkHr = -1
	backupsInstance.bkfolder = EnvironmentInstance.GetEnv("BK_PATH", "")
	backupsInstance.hourList = EnvironmentInstance.GetIntArray("BK_HOURS")

	path, err := filepath.Abs(backupsInstance.bkfolder)
	if err != nil {
		panic(err)
	}
	log.Printf("backup directory:%s", path)
	backupsInstance.bkfolder = path

	if _, err := os.Stat(backupsInstance.bkfolder); os.IsNotExist(err) {
		log.Printf("directory not found, %s, please create it first", backupsInstance.bkfolder)
		panic(err)
	}
}

func (b *Backups) run() {
	if EnvironmentInstance.GetBoolEnv("NOBACKUP") {
		return
	}
	for {
		time.Sleep(1 * time.Minute)

		if b.buckets.serverState == Stopped {
			return
		}

		b.runBk()

	}
}

func (b *Backups) createBackup(name BucketName, db *badger.DB) {

	suffix_day := EnvironmentInstance.GetBoolEnv("BK_SUFFIX_DAY")
	suffix_hour := EnvironmentInstance.GetBoolEnv("BK_SUFFIX_HOUR")
	bk_zip := EnvironmentInstance.GetBoolEnv("BK_ZIP")

	stats.backups[name].lastStart = time.Now()
	stats.backups[name].status = "running"
	stats.backups[name].lastMessage = ""

	// log.Printf("Backup started: %s\n", name)
	orig_bfname := createBackupFilename(name, suffix_day, suffix_hour)
	bfname := orig_bfname
	if bk_zip {
		bfname = addZipToFilename(bfname)
	}

	destFilename := path.Join(b.bkfolder, bfname)
	f, err := os.Create(destFilename)
	if err != nil {
		stats.backups[name].lastEnd = time.Now()
		stats.backups[name].status = "failed"
		stats.backups[name].lastMessage = "error creating backup: " + err.Error()
		return
	}

	var w io.Writer
	var wz *zip.Writer
	var wb *bufio.Writer

	if bk_zip {
		wz = zip.NewWriter(f)
		w, err = wz.Create(orig_bfname)
		if err != nil {
			stats.backups[name].status = "failed"
			stats.backups[name].lastMessage = "error creating zip file: " + err.Error()
			return
		}
	} else {
		wb = bufio.NewWriter(f)
		w = wb
	}

	_, err = db.Backup(w, 0)
	stats.backups[name].lastEnd = time.Now()
	if err != nil {
		stats.backups[name].status = "failed"
		stats.backups[name].lastMessage = "error creating backup: " + err.Error()
		return
	} else {
		stats.backups[name].status = "completed"
		stats.backups[name].lastMessage = ""
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
			//var wrap error
			//if err, ok := rec.(error); ok {
			//	wrap = errors.Wrap(err, "Panic Handler")
			//} else {
			//	wrap = errors.Errorf("%v", rec)
			//}
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

		for name, db := range buckets.dbBucket {
			b.createBackup(name, db)
		}
	}
}

func (b *Backups) sendSCP(name BucketName, fileToSend string) {
	scp_host := EnvironmentInstance.GetEnv("BK_SCP_HOST", "")
	scp_dir := EnvironmentInstance.GetEnv("BK_SCP_DIR", "")
	scp_uname := EnvironmentInstance.GetEnv("BK_SCP_UNAME", "")
	scp_upwd := EnvironmentInstance.GetEnv("BK_SCP_UPWD", "")
	scp_keypath := EnvironmentInstance.GetEnv("BK_SCP_PATH_TO_KEY", "")
	suffix_day := EnvironmentInstance.GetBoolEnv("BK_SCP_SUFFIX_DAY")
	suffix_hour := EnvironmentInstance.GetBoolEnv("BK_SCP_SUFFIX_HOUR")
	bk_zip := EnvironmentInstance.GetBoolEnv("BK_ZIP")

	if len(scp_host) == 0 ||
		len(scp_dir) == 0 ||
		len(scp_uname) == 0 ||
		(len(scp_upwd) == 0 &&
			len(scp_keypath) == 0) {
		return
	}

	stats.backups[name].status = "scp"

	scpDestName := createBackupFilename(name, suffix_day, suffix_hour)
	if bk_zip {
		scpDestName = addZipToFilename(scpDestName)
	}

	var sshConf *ssh.ClientConfig

	if len(scp_upwd) > 0 {
		log.Printf("scp using name/password %s. %s", scp_uname, strings.Repeat("x", len(scp_upwd)))
		sshConf = scp.NewSSHConfigFromPassword(scp_uname, scp_upwd)
	} else {
		log.Printf("scp using name/private key")
		privPEM, err := ioutil.ReadFile(scp_keypath)
		if err != nil {
			stats.backups[name].lastMessage = fmt.Sprintf("error creating scp config read private key %s", err.Error())
			stats.backups[name].status = "error"
			return
		}
		sshConf, err = scp.NewSSHConfigFromPrivateKey(scp_uname, privPEM)
		if err != nil {
			stats.backups[name].lastMessage = fmt.Sprintf("error creating scp config with private key %s", err.Error())
			stats.backups[name].status = "error"
			return
		}

	}
	scpClient, err := scp.NewClient(scp_host, sshConf, &scp.ClientOption{})
	if err != nil {
		stats.backups[name].lastMessage = fmt.Sprintf("error creating scp client %s", err.Error())
		stats.backups[name].status = "error"
		return
	}
	defer scpClient.Close()

	transferOptions := &scp.FileTransferOption{
		Context:      context.Background(),
		Timeout:      30 * time.Second,
		PreserveProp: true,
	}
	destFile := path.Join(scp_dir, scpDestName)
	log.Printf("Scp %s:%s -> %s", scp_host, fileToSend, destFile)
	err = scpClient.CopyFileToRemote(fileToSend, destFile, transferOptions)
	if err != nil {
		log.Printf("error sending file:%s, %s", name, err)
		stats.backups[name].lastMessage = fmt.Sprintf("error during send %s", err.Error())
		stats.backups[name].status = "error"
		return
	}

	stats.backups[name].lastEnd = time.Now()
	stats.backups[name].status = "completed"
}
