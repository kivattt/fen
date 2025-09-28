//go:build !windows && !linux

package gogitstatus

import (
	"os"
	"syscall"
)

func isCTimeUnchanged(stat os.FileInfo, mTimeSec, mTimeNsec int64) bool {
	unixStat := stat.Sys().(*syscall.Stat_t)
	return int64(unixStat.Ctimespec.Sec) == mTimeSec && int64(unixStat.Ctimespec.Nsec) == mTimeNsec
}
