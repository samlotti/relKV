package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"relKV/common"
	"strings"
	"syscall"
	"time"
)

var BucketsInstance *BucketsDb

func BootServer(version string, readyChannel chan *BucketsDb) {
	log.Printf("Starting relKV %s\n", version)
	EnvInit()

	logFile := EnvironmentInstance.GetEnv("LOG_FILE", "")

	listen := EnvironmentInstance.GetEnv("HTTP_HOST", "0.0.0.0:8080")
	if !strings.Contains(listen, ":") {
		panic("HTTP_HOST should contain a port")
	}

	// Make sure port is available

	if !CheckPortAvail(listen) {
		log.Fatal(fmt.Sprintf("port in use: %s", listen))
	}

	if len(logFile) > 0 {
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)
	}

	go ProcessUnixCommands()

	BucketsInstance = &BucketsDb{
		version:        version,
		logfile:        logFile,
		listenAddrPort: listen,
		ServerState:    Starting,
		baseTableSize:  8 << 20, // 8MB
		dbPath:         EnvironmentInstance.GetEnv("DB_PATH", ""),
		buckets:        EnvironmentInstance.GetBucketArray("BUCKETS"),
		allowCreate:    EnvironmentInstance.GetBoolEnv("ALLOW_CREATE_DB"),
		Jobs:           make([]*common.ScpJob, 0),
	}

	BucketsInstance.Init()
	StatsInstance.init()

	BucketsInstance.openDBBuckets()

	defer BucketsInstance.Close()

	go BucketsInstance.runGC()

	//cmd.BackupsInit(BucketsInstance)
	//go cmd.BackupsInstance.Run()

	log.Printf("Listening on:%s", listen)

	srv := http.Server{
		Addr:              listen,
		Handler:           BucketsInstance.newHTTPRouter(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		BucketsInstance.stopChan = make(chan os.Signal, 1)

		signal.Notify(BucketsInstance.stopChan, os.Interrupt, syscall.SIGTERM)
		sig := <-BucketsInstance.stopChan
		log.Println("shutdown:", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP server shutdown failed:%+s", err)
		}
		BucketsInstance.ServerState = Stopped
	}()

	BucketsInstance.ServerState = Running
	readyChannel <- BucketsInstance

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}

}

func ProcessUnixCommands() {
	unixSocket := EnvironmentInstance.GetEnv("CMD_UNIX_SOCKET", "")
	if len(unixSocket) == 0 {
		log.Println("No socket support, CMD_UNIX_SOCKET not specified")
		return
	}

	_, err := os.Stat(unixSocket)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		// log.Fatal("sock file exists: ", unixSocket)
		os.Remove(unixSocket)
	}

	l, err := net.Listen("unix", unixSocket)
	if err != nil {
		log.Fatal("cannot open:", unixSocket, " err: ", err)
	}

	for {
		fd, err := l.Accept()
		if err != nil {
			log.Println("accept error on unix socket:", err)
			continue
		}
		go handleCommands(fd)
	}
}

func handleCommands(fd net.Conn) {
	fmt.Println("unix client connected")
	buf := make([]byte, 1024)
	n, err := fd.Read(buf[:])
	if err != nil {
		return
	}
	cmd := string(buf[:n])
	if cmd == "stop\n" {
		fmt.Println("Stop called")
		BucketsInstance.shutDownServer()
		fd.Write([]byte("stopped\n"))
	}

	fd.Close()
}

// CheckPortAvail if a port is available
func CheckPortAvail(port string) bool {

	// Try to create a server with the port
	server, err := net.Listen("tcp", port)

	// if it fails then the port is likely taken
	if err != nil {
		return false
	}

	// close the server
	server.Close()

	// we successfully used and closed the port
	// so it's now available to be used again
	return true

}
