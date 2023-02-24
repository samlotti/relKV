package backup

import (
	"fmt"
	"kvDb/cmd"
	"time"
)

func CreateBackupFilename(name cmd.BucketName, addDay bool, addHour bool) string {
	n := string(name)
	if addDay {
		n = fmt.Sprintf("%s_%02d", n, time.Now().Day())
	}
	if addHour {
		n = fmt.Sprintf("%s_%02d", n, time.Now().Hour())
	}
	return n + ".bak"
}

func AddZipToFilename(name string) string {
	return name + ".zip"
}
