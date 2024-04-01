package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rivo/tview"
)

func PathWithEndSeparator(path string) string {
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return path
	}

	return path + string(os.PathSeparator)
}

func PathWithoutEndSeparator(path string) string {
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return path[:len(path)-1] // os.PathSeparator is a rune, so always 1 character long
	}

	return path
}

// TODO: Maybe make these file functions take a fs.FileInfo from a previously done os.Stat()

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

func FilePermissionsString(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	var ret strings.Builder

	permissionChars := "xwr"
	for i := 8; i >= 0; i-- {
		if stat.Mode()>>i&1 == 1 {
			ret.WriteByte(permissionChars[i%3])
		} else {
			ret.WriteByte('-')
		}
	}

	return ret.String(), nil
}

func FileLastModifiedString(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	return stat.ModTime().Format(time.UnixDate), nil
}

func OpenFile(path string, app *tview.Application) {
	suffixProgramMap := map[string]string{
		".mp4":  "mpv",
		".mp3":  "mpv",
		".wav":  "mpv",
		".flac": "mpv",
		".mov":  "mpv",
		".webm": "mpv",

		// feh breaks terminal on close
		/*".png":  "feh",
		".jpg":  "feh",
		".jpeg": "feh",
		".jfif": "feh",
		".flif": "feh",
		".tiff": "feh",
		".gif":  "feh",
		".webp": "feh",*/
	}

	programFallBacks := []string{"nvim", "vim", "vi", "nano"}

	for key, value := range suffixProgramMap {
		if strings.HasSuffix(path, key) {
			programFallBacks = append([]string{value}, programFallBacks...)
			break
		}
	}

	app.Suspend(func() {
		var err error
		for _, program := range programFallBacks {
			cmd := exec.Command(program, path)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()
			if err == nil {
				break
			}
		}

		if err != nil {
			log.Fatal(err)
		}
	})
}
