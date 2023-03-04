package backup

import (
	"fmt"
	"relKV/common"
	"time"
)

func CreateBackupFilename(name common.BucketName, addDay bool, addHour bool) string {
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
