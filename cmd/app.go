package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type BucketName string

var buckets *BucketsDb

func BootServer(version string) {
	log.Printf("Starting kvDb %s\n", version)
	EnvInit()

	listen := Environment.GetEnv("HTTP_HOST", "0.0.0.0:8080")

	// Make sure port is available
	if !CheckPortAvail(listen) {
		panic(fmt.Sprintf("port in use: %s", listen))
	}

	f, err := os.OpenFile("kvDb.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	buckets = &BucketsDb{
		baseTableSize: 8 << 20, // 8MB
		dbPath:        Environment.GetEnv("DB_PATH", "/var/lib/bucketsDB"),
		buckets:       Environment.GetBucketArray("BUCKETS"),
	}

	buckets.Init()
	stats.init()

	buckets.openDBBuckets()

	defer buckets.Close()

	go buckets.runGC()

	BackupsInit()
	go Backups.run()

	log.Printf("Listening on:%s", listen)

	srv := http.Server{
		Addr:              listen,
		Handler:           buckets.newHTTPRouter(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		signChan := make(chan os.Signal, 1)

		signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)
		sig := <-signChan
		log.Println("shutdown:", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP server shutdown failed:%+s", err)
		}
	}()

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
