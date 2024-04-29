package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
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

func HasSuffixFromList(str string, list []string) bool {
	for _, e := range list {
		if strings.HasSuffix(str, e) {
			return true
		}
	}

	return false
}

func FileColor(path string) tcell.Color {
	imageTypes := []string{
		".png",
		".jpg",
		".jpeg",
		".jfif",
		".flif",
		".tiff",
		".gif",
		".webp",
		".bmp",
	}

	videoTypes := []string{
		".mp4",
		".webm",
		".mkv",
		".mov",
		".avi",
		".flv",
	}

	audioTypes := []string{
		".wav",
		".flac",
		".mp3",
		".ogg",
		".m4a",
	}

	archiveTypes := []string{
		".zip",
		".jar",
		".kra",

		// https://en.wikipedia.org/wiki/Tar_(computing)
		".tar.bz2", ".tb2", ".tbz", ".tbz2", ".tz2",
		".tar.gz", ".taz", ".tgz",
		".tar.lz",
		".tar.lzma", ".tlz",
		".tar.lzo",
		".tar.xz", ".tZ", ".taZ",
		".tar.zst", ".tzst",
	}

	codeTypes := []string{
		".go",
		".cpp",
		".cxx",
		".hpp",
		".hxx",
		".h",
		".c",
		".cc",
		".py",
		".sh",
		".bash",
		".js",
		".jsx",
		".ts",
		".tsx",
		".rs",
		".lua",
		".vim",
		".java",
	}

	documentTypes := []string{
		".md",
		".pdf",
		".epub",
		".docx",
		".doc",
		".odg",
		".fodg",
		".otg",
	}

	if HasSuffixFromList(path, imageTypes) {
		return tcell.ColorYellow
	}

	if HasSuffixFromList(path, videoTypes) {
		return tcell.ColorHotPink
	}

	if HasSuffixFromList(path, archiveTypes) {
		return tcell.ColorRed
	}

	if HasSuffixFromList(path, codeTypes) {
		return tcell.ColorAqua
	}

	if HasSuffixFromList(path, audioTypes) {
		return tcell.ColorPurple
	}

	if HasSuffixFromList(path, documentTypes) {
		return tcell.ColorGray
	}

	return tcell.ColorDefault
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
		".webp": "feh",
		".bmp": "feh",*/
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
