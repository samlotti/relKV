package commands

import (
	"archive/zip"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"relKV/cmd"
	"strings"
)

func handleRestore(cmds []string) {
	fmt.Println("restore")

	if len(cmds) != 3 {
		fmt.Println("Expected restore {fromFile} {toDbName}")
		handleHelp()
		os.Exit(12)
	}

	fname := cmds[1]
	dest := cmds[2]

	fmt.Printf("from %s - %s\n", fname, dest)

	bzip := strings.HasSuffix(fname, ".zip")

	if bzip {
		reader, err := zip.OpenReader(fname)
		defer reader.Close()
		if err != nil {
			log.Printf("error opening zip file: %s", err.Error())
			os.Exit(12)
		}
		if len(reader.File) != 1 {
			log.Println("too many files in the zip, please unzip and then restore the individual file", err.Error())
			os.Exit(12)
		}
		log.Printf("processing zip content file: %s", reader.File[0].Name)
		srcFile, err := reader.File[0].Open()
		if err != nil {
			log.Printf("error opening zip content file: %s ->  %s", reader.File[0].Name, err.Error())
			os.Exit(12)
		}
		defer srcFile.Close()
		do_restore(srcFile, dest)

	} else {
		// Non zip file
		srcFile, err := os.Open(fname)
		if err != nil {
			log.Printf("error opening file: %s", err.Error())
			os.Exit(12)
		}
		defer srcFile.Close()
		do_restore(srcFile, dest)
	}
}

func do_restore(file io.ReadCloser, dest string) {
	dbPath := cmd.EnvironmentInstance.GetEnv("DB_PATH", "")
	if len(dbPath) == 0 {
		log.Printf("dbpath not specified")
		os.Exit(12)
	}
	sstDir := filepath.Join(dbPath, dest)
	manifestFile := filepath.Join(sstDir, badger.ManifestFilename)
	if _, err := os.Stat(manifestFile); err == nil { // No error. File already exists.
		log.Printf("cannot restore to an already existing database")
		os.Exit(12)
	} else if os.IsNotExist(err) {
		// pass
	} else { // Return an error if anything other than the error above
		log.Printf("cannot stat file: %s", manifestFile)
		os.Exit(12)
	}

	// Open DB
	db, err := badger.Open(badger.DefaultOptions(sstDir).
		WithValueDir(sstDir).
		WithNumVersionsToKeep(math.MaxInt32))
	if err != nil {
		log.Printf("error opening db: %s", err)
		os.Exit(12)
	}
	defer db.Close()

	// Open File

	// Run restore
	err = db.Load(file, 256)
	if err != nil {
		log.Printf("had an error load db: %s", err)
		os.Exit(12)
	} else {
		log.Printf("database %s restored", dest)
	}
}
