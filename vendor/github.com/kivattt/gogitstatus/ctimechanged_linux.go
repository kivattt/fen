//go:build linux

package gogitstatus

import (
	"os"
	"syscall"
)

func isCTimeUnchanged(stat os.FileInfo, mTimeSec, mTimeNsec int64) bool {
	unixStat := stat.Sys().(*syscall.Stat_t)
	return int64(unixStat.Ctim.Sec) == mTimeSec && int64(unixStat.Ctim.Nsec) == mTimeNsec
}
