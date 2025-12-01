//go:build windows

package gogitstatus

import (
	"os"
)

func isCTimeUnchanged(stat os.FileInfo, mTimeSec, mTimeNsec int64) bool {
	return true
}
