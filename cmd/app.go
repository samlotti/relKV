package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type BucketName string

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
		panic(fmt.Sprintf("port in use: %s", listen))
	}

	if len(logFile) > 0 {
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)
	}

	BucketsInstance = &BucketsDb{
		logfile:        logFile,
		listenAddrPort: listen,
		ServerState:    Starting,
		baseTableSize:  8 << 20, // 8MB
		dbPath:         EnvironmentInstance.GetEnv("DB_PATH", ""),
		buckets:        EnvironmentInstance.GetBucketArray("BUCKETS"),
		allowCreate:    EnvironmentInstance.GetBoolEnv("ALLOW_CREATE_DB"),
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

	log.Println("sending ready")
	BucketsInstance.ServerState = Running
	readyChannel <- BucketsInstance
	log.Println("sent ready")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}

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
