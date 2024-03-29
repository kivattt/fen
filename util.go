package main

import (
	"errors"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

func EntrySize(path string, ignoreHiddenFiles bool) (string, error) {
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

		if ignoreHiddenFiles {
			withoutHiddenFiles := []os.DirEntry{}
			for _, e := range files {
				if !strings.HasPrefix(e.Name(), ".") {
					withoutHiddenFiles = append(withoutHiddenFiles, e)
				}
			}

			files = withoutHiddenFiles
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
