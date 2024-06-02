package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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

func StyleToStyleTagString(style tcell.Style) string {
	foreground, background, attributeMask := style.Decompose()
	// https://pkg.go.dev/github.com/gdamore/tcell/v2#AttrMask
	attributeStyleTagNameLookup := "blrudis"
	attributesString := ""
	if attributeMask != 0 {
		for i := 0; i < len(attributeStyleTagNameLookup); i++ {
			if attributeMask&(1<<i) != 0 {
				attributesString += string(attributeStyleTagNameLookup[i])
			}
		}
	}

	if attributesString == "" {
		return "[" + foreground.String() + ":" + background.String() + "]"
	}

	return "[" + foreground.String() + ":" + background.String() + ":" + attributesString + "]"
}

func FileColor(path string) tcell.Style {
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
		".txt",
	}

	var ret tcell.Style

	stat, err := os.Stat(path)
	if err == nil {
		if stat.IsDir() {
			return ret.Foreground(tcell.ColorBlue).Bold(true)
		} else if stat.Mode().IsRegular() {
			if stat.Mode()&0111 != 0 {
				return ret.Foreground(tcell.NewRGBColor(0, 255, 0)).Bold(true) // Green
			}
		} else {
			return ret.Foreground(tcell.ColorDarkGray)
		}
	}

	if HasSuffixFromList(path, imageTypes) {
		return ret.Foreground(tcell.ColorYellow)
	}

	if HasSuffixFromList(path, videoTypes) {
		return ret.Foreground(tcell.ColorHotPink)
	}

	if HasSuffixFromList(path, archiveTypes) {
		return ret.Foreground(tcell.ColorRed)
	}

	if HasSuffixFromList(path, codeTypes) {
		return ret.Foreground(tcell.ColorAqua)
	}

	if HasSuffixFromList(path, audioTypes) {
		return ret.Foreground(tcell.ColorPurple)
	}

	if HasSuffixFromList(path, documentTypes) {
		return ret.Foreground(tcell.ColorGray)
	}

	return ret.Foreground(tcell.ColorDefault)
}

// We could maybe cache this to a certain extent
func ProgramsAndDescriptionsForFile(fen *Fen) ([]string, []string) {
	matched := false
	var programs []string
	var descriptions []string
	for _, programMatch := range fen.config.OpenWith {
		for _, match := range programMatch.Match {
			matched, _ = filepath.Match(match, filepath.Base(fen.sel))
			if matched {
				break
			}
		}

		if matched {
			for _, program := range programMatch.Programs {
				programs = append(programs, program)
				descriptions = append(descriptions, "User config")
			}
			break
		}
	}

	if runtime.GOOS == "darwin" { // macOS
		programs = append(programs, "open")
		descriptions = append(descriptions, "macOS")
	} else if runtime.GOOS == "windows" {
		// TODO: Use the rundll32.exe FileProtocolHandler thing
		programs = append(programs, "notepad")
		descriptions = append(descriptions, "Windows")
	} else {
		programs = append(programs, "xdg-open")
		descriptions = append(descriptions, "Linux")
	}

	editor := os.Getenv("EDITOR")
	if editor != "" {
		programs = append(programs, editor)
		descriptions = append(descriptions, "$EDITOR")
	}
	programs = append(programs, "vim -p", "vi -p", "nano")
	for i := 0; i < 3; i++ {
		descriptions = append(descriptions, "Default fallback")
	}

	// Remove duplicate programs
	i := 0
	for j, program := range programs {
		if !slices.Contains(programs[:i], program) {
			programs[i] = program
			descriptions[i] = descriptions[j]
			i++
		}
	}

	programs = programs[:i]
	descriptions = descriptions[:i]

	return programs, descriptions
}

func OpenFile(fen *Fen, app *tview.Application, openWith string) {
	if fen.config.PrintPathOnOpen && openWith == "" {
		app.Stop()
		fmt.Println(fen.sel)
		return
	}

	programsAndFallbacks, _ := ProgramsAndDescriptionsForFile(fen)
	if openWith != "" {
		programsAndFallbacks = append([]string{openWith}, programsAndFallbacks...)
	}

	app.Suspend(func() {
		for _, program := range programsAndFallbacks {
			programSplitSpace := strings.Split(program, " ")

			programName := programSplitSpace[0]
			programArguments := []string{}
			if len(programSplitSpace) > 1 {
				programArguments = programSplitSpace[1:]
			}

			var cmd *exec.Cmd
			if len(fen.selected) <= 0 {
				cmd = exec.Command(programName, append(programArguments, fen.sel)...)
			} else {
				cmd = exec.Command(programName, append(programArguments, fen.selected...)...)
			}
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

func FoldersAtBeginning(dirEntries []os.DirEntry) []os.DirEntry {
	var folders []os.DirEntry
	var files []os.DirEntry
	for _, entry := range dirEntries {
		if entry.IsDir() {
			folders = append(folders, entry)
		} else {
			files = append(files, entry)
		}
	}

	if len(folders)+len(files) != len(dirEntries) {
		panic("FoldersAtBeginning failed!")
	}

	return append(folders, files...)
}

func FilePathUniqueNameIfAlreadyExists(path string) string {
	if path != filepath.Clean(path) {
		panic("FilePathUniqueNameIfAlreadyExists got an uncleaned file path")
	}

	newPath := path
	for i := -1;; i++ {
		_, err := os.Stat(newPath)
		if err != nil {
			return newPath
		}

		if i == -1 {
			newPath = path + "_"
		} else {
			newPath = path + "_" + strconv.Itoa(i)
		}
	}
}
