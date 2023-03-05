package backup

import (
	"context"
	"fmt"
	"github.com/povsister/scp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"path"
	. "relKV/cmd"
	"relKV/common"
	"runtime/debug"
	"sync"
	"time"
)

type ScpEnv struct {
	scpHost    string
	scpDir     string
	scpUname   string
	scpUpwd    string
	scpKeypath string
	suffixDay  bool
	suffixHour bool
	bkZip      bool

	mutex sync.Mutex

	buckets *BucketsDb
}

var ScpEnvInstance *ScpEnv

func ScpInit(b *BucketsDb) {
	ScpEnvInstance = &ScpEnv{}
	ScpEnvInstance._init(b)
	go ScpEnvInstance.SendLoop()
}

func (s *ScpEnv) _init(b *BucketsDb) {
	s.scpHost = EnvironmentInstance.GetEnv("BK_SCP_HOST", "")
	s.scpDir = EnvironmentInstance.GetEnv("BK_SCP_DIR", "")
	s.scpUname = EnvironmentInstance.GetEnv("BK_SCP_UNAME", "")
	s.scpUpwd = EnvironmentInstance.GetEnv("BK_SCP_UPWD", "")
	s.scpKeypath = EnvironmentInstance.GetEnv("BK_SCP_PATH_TO_KEY", "")
	s.suffixDay = EnvironmentInstance.GetBoolEnv("BK_SCP_SUFFIX_DAY")
	s.suffixHour = EnvironmentInstance.GetBoolEnv("BK_SCP_SUFFIX_HOUR")
	s.bkZip = EnvironmentInstance.GetBoolEnv("BK_ZIP")
	s.buckets = b
}

func (s *ScpEnv) IsEnabled() bool {
	if len(s.scpHost) == 0 ||
		len(s.scpDir) == 0 ||
		len(s.scpUname) == 0 ||
		(len(s.scpUpwd) == 0 &&
			len(s.scpKeypath) == 0) {
		return false
	}
	return true
}

func (s *ScpEnv) SendLoop() {
	for {
		time.Sleep(15 * time.Second)

		if s.buckets.ServerState == Stopped {
			return
		}

		s.selectAndRunAJob()
		// log.Printf("scp done")
	}
}

func (s *ScpEnv) AddScpJob(bname common.BucketName, filename string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, job := range s.buckets.Jobs {
		if job.BucketName == bname {
			if job.Status == common.ScpComplete || job.Status == common.ScpError {
				job.Status = common.ScpPending
				job.NextSend = time.Now()
			}
			return
		}
	}

	s.buckets.Jobs = append(s.buckets.Jobs, &common.ScpJob{
		Fname:      filename,
		BucketName: bname,
		Status:     common.ScpPending,
		Message:    "",
		LastStart:  time.Now(),
		LastEnd:    time.Now(),
		NextSend:   time.Now(),
	})

}

func (s *ScpEnv) selectAndRunAJob() {
	j := s.selectAJob()
	if j != nil {
		s.processJob(j)
	}
}

func (s *ScpEnv) selectAJob() *common.ScpJob {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// log.Printf("scp selecting a job")
	j := &common.ScpJob{NextSend: time.Time{}, Fname: ""}
	for _, job := range s.buckets.Jobs {
		if job.Status == common.ScpPending || job.Status == common.ScpError {
			if time.Now().After(job.NextSend) {
				if j.Fname == "" || j.NextSend.After(job.NextSend) {
					j = job
				}

			}
		}
	}

	//log.Printf("scp job %s", j.Fname)
	if j.Fname == "" {
		return nil
	}
	return j

}

func (s *ScpEnv) processJob(j *common.ScpJob) {
	// Don't let it die, try again next time
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Println("error in scp", rec)
			fmt.Printf("%s", debug.Stack())

			j.Status = common.ScpError
			j.NextSend = time.Now().Add(5 * time.Minute)
		}
		j.LastEnd = time.Now()
	}()
	s.sendScp(j)

}

func (s *ScpEnv) sendScp(j *common.ScpJob) {
	j.Status = common.ScpRunning
	j.LastStart = time.Now()

	scpDestName := CreateBackupFilename(j.BucketName, ScpEnvInstance.suffixDay, ScpEnvInstance.suffixHour)
	if ScpEnvInstance.bkZip {
		scpDestName = AddZipToFilename(scpDestName)
	}

	var sshConf *ssh.ClientConfig

	if len(ScpEnvInstance.scpUpwd) > 0 {
		// log.Printf("scp using name/password %s. %s", ScpEnvInstance.scpUname, strings.Repeat("x", len(ScpEnvInstance.scpUpwd)))
		sshConf = scp.NewSSHConfigFromPassword(ScpEnvInstance.scpUname, ScpEnvInstance.scpUpwd)
	} else {
		// log.Printf("scp using name/private key")
		privPEM, err := ioutil.ReadFile(ScpEnvInstance.scpKeypath)
		if err != nil {
			j.Message = fmt.Sprintf("error creating scp config read private key %s", err.Error())
			j.Status = common.ScpError
			j.NextSend = time.Now().Add(5 * time.Minute)
			return
		}
		sshConf, err = scp.NewSSHConfigFromPrivateKey(ScpEnvInstance.scpUname, privPEM)
		if err != nil {
			j.Message = fmt.Sprintf("error creating scp config with private key %s", err.Error())
			j.Status = common.ScpError
			j.NextSend = time.Now().Add(5 * time.Minute)
			return
		}

	}
	scpClient, err := scp.NewClient(ScpEnvInstance.scpHost, sshConf, &scp.ClientOption{})
	if err != nil {
		j.Message = fmt.Sprintf("error creating scp client %s", err.Error())
		j.Status = common.ScpError
		j.NextSend = time.Now().Add(5 * time.Minute)
		return
	}
	defer scpClient.Close()

	transferOptions := &scp.FileTransferOption{
		Context:      context.Background(),
		Timeout:      0,
		PreserveProp: true,
	}
	destFile := path.Join(ScpEnvInstance.scpDir, scpDestName)
	// log.Printf("Scp %s:%s -> %s", ScpEnvInstance.scpHost, j.Fname, destFile)
	err = scpClient.CopyFileToRemote(j.Fname, destFile, transferOptions)
	if err != nil {
		log.Printf("error sending file:%s, %s", j.Fname, err)
		j.Message = fmt.Sprintf("error during send %s", err.Error())
		j.Status = common.ScpError
		j.NextSend = time.Now().Add(5 * time.Minute)
		return
	}

	j.Status = common.ScpComplete

}
