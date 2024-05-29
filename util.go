package main

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Trims the last decimals up to maxDecimals, does nothing if maxDecimals is less than 0, e.g -1
func trimLastDecimals(numberString string, maxDecimals int) string {
	if maxDecimals < 0 {
		return numberString
	}

	dotIndex := strings.Index(numberString, ".")
	if dotIndex == -1 {
		return numberString
	}

	return numberString[:min(len(numberString), dotIndex+maxDecimals+1)]
}

// If maxDecimals is less than 0, e.g -1, we show the exact size down to the byte
// https://en.wikipedia.org/wiki/Byte#Multiple-byte_units
func BytesToHumanReadableUnitString(bytes uint64, maxDecimals int) string {
	unitValues := []float64{
		math.Pow(10, 3),
		math.Pow(10, 6),
		math.Pow(10, 9),
		math.Pow(10, 12),
		math.Pow(10, 15),
		math.Pow(10, 18), // Largest unit that fits in 64 bits
	}

	unitStrings := []string{
		"kB",
		"MB",
		"GB",
		"TB",
		"PB",
		"EB",
	}

	if bytes < uint64(unitValues[0]) {
		return strconv.FormatUint(bytes, 10) + " B"
	}

	for i, v := range unitValues {
		if bytes >= uint64(v) {
			continue
		}

		lastIndex := max(0, i-1)
		return trimLastDecimals(strconv.FormatFloat(float64(bytes)/unitValues[lastIndex], 'f', -1, 64), maxDecimals) + " " + unitStrings[lastIndex]
	}

	return trimLastDecimals(strconv.FormatFloat(float64(bytes)/unitValues[len(unitValues)-1], 'f', -1, 64), maxDecimals) + " " + unitStrings[len(unitStrings)-1]
}

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
		return BytesToHumanReadableUnitString(uint64(stat.Size()), 2), nil
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

func OpenFile(fen *Fen, app *tview.Application) {
	matched := false
	var programsToUse []string
	for _, programMatch := range fen.config.OpenWith {
		for _, match := range programMatch.Match {
			matched, _ = filepath.Match(match, filepath.Base(fen.sel))
			if matched {
				break
			}
		}

		if matched {
			programsToUse = programMatch.Programs
			break
		}
	}

	var programsAndFallbacks []string
	if runtime.GOOS == "darwin" { // macOS
		programsAndFallbacks = []string{"open"}
	} else if runtime.GOOS == "windows" {
		// TODO: Use the rundll32.exe FileProtocolHandler thing
		programsAndFallbacks = []string{"notepad"}
	} else {
		programsAndFallbacks = []string{"xdg-open"}
		editor := os.Getenv("EDITOR")
		if editor != "" {
			programsAndFallbacks = append(programsAndFallbacks, editor)
		}
		programsAndFallbacks = append(programsAndFallbacks, "vim", "vi", "nano")
	}

	if matched {
		programsAndFallbacks = append(programsToUse, programsAndFallbacks...)
	}

	app.Suspend(func() {
		for _, program := range programsAndFallbacks {
			cmd := exec.Command(program, fen.sel)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err == nil {
				break
			}
		}
	})
}
