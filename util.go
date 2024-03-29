package main

import (
	"errors"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func EntrySize(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return strconv.FormatInt(stat.Size(), 10) + " B", nil
	} else {
		files, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(len(files)), nil
	}
}

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

	username, _ := user.LookupId(strconv.Itoa(uid))
	groupname, _ := user.LookupGroupId(strconv.Itoa(gid))

	return username.Username, groupname.Name, nil
}
