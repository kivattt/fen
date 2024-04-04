//go:build !windows
// +build !windows

package main

import (
	"errors"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func FileUserAndGroupName(path string) (string, string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}

	syscallStat, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return "", "", errors.New("Unable to syscall stat")
	}

	uid := int(syscallStat.Uid)
	gid := int(syscallStat.Gid)

	username, usernameErr := user.LookupId(strconv.Itoa(uid))
	groupname, groupnameErr := user.LookupGroupId(strconv.Itoa(gid))

	usernameStr := ""
	groupnameStr := ""

	if usernameErr == nil {
		usernameStr = username.Username
	}

	if groupnameErr == nil {
		groupnameStr = groupname.Name
	}

	return usernameStr, groupnameStr, nil
}
