package main

//lint:file-ignore ST1005 some user-visible messages are stored in error values and thus occasionally require capitalization

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charlievieth/strcase"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/term"
	luar "layeh.com/gopher-luar"
)

const pressAnyKeyToContinueText = "Press any key to continue..."
const pressEnterToContinueText = "Press Enter to continue..."

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

func BytesToFileSizeFormat(bytes uint64, maxDecimals int, format string) string {
	if format == HUMAN_READABLE {
		return BytesToHumanReadableUnitString(bytes, maxDecimals)
	} else if format == BYTES {
		return strconv.FormatUint(bytes, 10) + " B"
	} else {
		panic("Invalid file_size_format value \"" + format + "\"")
	}
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

// stat should be from an os.Lstat(). If stat is nil, it returns an error.
func EntrySizeText(folderFileCountCache map[string]int, stat os.FileInfo, path string, hiddenFiles bool, format string) (string, error) {
	if stat == nil {
		return "", errors.New("stat was nil")
	}

	var ret strings.Builder

	// Show the size of the target, not the symlink
	// TODO: Use filepath.EvalSymlinks() ?
	if stat.Mode()&os.ModeSymlink != 0 {
		var err error
		stat, err = os.Stat(path)
		if err != nil {
			return "", err
		}
	}

	if !stat.IsDir() {
		ret.WriteString(BytesToFileSizeFormat(uint64(stat.Size()), 2, format))
	} else {
		count, err := FolderFileCountCached(folderFileCountCache, path, hiddenFiles)
		if err != nil {
			return "", err
		}

		ret.WriteString(strconv.Itoa(count))
	}

	return ret.String(), nil
}

func FolderFileCountCached(cache map[string]int, path string, hiddenFiles bool) (int, error) {
	if cache == nil {
		panic("In FolderFileCountCached(): cache (fen.folderFileCountCache) was nil")
	}

	count, ok := cache[path]
	if ok {
		return count, nil
	}

	var err error
	count, err = FolderFileCount(path, hiddenFiles)
	if err != nil {
		return 0, err
	}

	cache[path] = count
	return count, nil
}

func FolderFileCount(path string, hiddenFiles bool) (int, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	if !hiddenFiles {
		withoutHiddenFiles := []os.DirEntry{}
		for _, e := range files {
			if !strings.HasPrefix(e.Name(), ".") {
				withoutHiddenFiles = append(withoutHiddenFiles, e)
			}
		}

		files = withoutHiddenFiles
	}

	return len(files), nil
}

func FilePermissionsString(stat os.FileInfo) string {
	var ret strings.Builder
	ret.Grow(8) // The output will be 8 bytes long

	permissionChars := "xwr"
	for i := 8; i >= 0; i-- {
		if stat.Mode()>>i&1 == 1 {
			ret.WriteByte(permissionChars[i%3])
		} else {
			ret.WriteByte('-')
		}
	}

	return ret.String()
}

func FileLastModifiedString(stat os.FileInfo) string {
	return stat.ModTime().Format(time.UnixDate)
}

// This map inverted: https://github.com/gdamore/tcell/blob/88b9c25c3c5ee48b611dfeca9a2e9cf07812c35e/color.go#L851
var colorToNamesMap = map[tcell.Color]string{
	tcell.ColorBlack:                "black",
	tcell.ColorMaroon:               "maroon",
	tcell.ColorGreen:                "green",
	tcell.ColorOlive:                "olive",
	tcell.ColorNavy:                 "navy",
	tcell.ColorPurple:               "purple",
	tcell.ColorTeal:                 "teal",
	tcell.ColorSilver:               "silver",
	tcell.ColorGray:                 "gray",
	tcell.ColorRed:                  "red",
	tcell.ColorLime:                 "lime",
	tcell.ColorYellow:               "yellow",
	tcell.ColorBlue:                 "blue",
	tcell.ColorFuchsia:              "fuchsia",
	tcell.ColorAqua:                 "aqua",
	tcell.ColorWhite:                "white",
	tcell.ColorAliceBlue:            "aliceblue",
	tcell.ColorAntiqueWhite:         "antiquewhite",
	tcell.ColorAquaMarine:           "aquamarine",
	tcell.ColorAzure:                "azure",
	tcell.ColorBeige:                "beige",
	tcell.ColorBisque:               "bisque",
	tcell.ColorBlanchedAlmond:       "blanchedalmond",
	tcell.ColorBlueViolet:           "blueviolet",
	tcell.ColorBrown:                "brown",
	tcell.ColorBurlyWood:            "burlywood",
	tcell.ColorCadetBlue:            "cadetblue",
	tcell.ColorChartreuse:           "chartreuse",
	tcell.ColorChocolate:            "chocolate",
	tcell.ColorCoral:                "coral",
	tcell.ColorCornflowerBlue:       "cornflowerblue",
	tcell.ColorCornsilk:             "cornsilk",
	tcell.ColorCrimson:              "crimson",
	tcell.ColorDarkBlue:             "darkblue",
	tcell.ColorDarkCyan:             "darkcyan",
	tcell.ColorDarkGoldenrod:        "darkgoldenrod",
	tcell.ColorDarkGray:             "darkgray",
	tcell.ColorDarkGreen:            "darkgreen",
	tcell.ColorDarkKhaki:            "darkkhaki",
	tcell.ColorDarkMagenta:          "darkmagenta",
	tcell.ColorDarkOliveGreen:       "darkolivegreen",
	tcell.ColorDarkOrange:           "darkorange",
	tcell.ColorDarkOrchid:           "darkorchid",
	tcell.ColorDarkRed:              "darkred",
	tcell.ColorDarkSalmon:           "darksalmon",
	tcell.ColorDarkSeaGreen:         "darkseagreen",
	tcell.ColorDarkSlateBlue:        "darkslateblue",
	tcell.ColorDarkSlateGray:        "darkslategray",
	tcell.ColorDarkTurquoise:        "darkturquoise",
	tcell.ColorDarkViolet:           "darkviolet",
	tcell.ColorDeepPink:             "deeppink",
	tcell.ColorDeepSkyBlue:          "deepskyblue",
	tcell.ColorDimGray:              "dimgray",
	tcell.ColorDodgerBlue:           "dodgerblue",
	tcell.ColorFireBrick:            "firebrick",
	tcell.ColorFloralWhite:          "floralwhite",
	tcell.ColorForestGreen:          "forestgreen",
	tcell.ColorGainsboro:            "gainsboro",
	tcell.ColorGhostWhite:           "ghostwhite",
	tcell.ColorGold:                 "gold",
	tcell.ColorGoldenrod:            "goldenrod",
	tcell.ColorGreenYellow:          "greenyellow",
	tcell.ColorHoneydew:             "honeydew",
	tcell.ColorHotPink:              "hotpink",
	tcell.ColorIndianRed:            "indianred",
	tcell.ColorIndigo:               "indigo",
	tcell.ColorIvory:                "ivory",
	tcell.ColorKhaki:                "khaki",
	tcell.ColorLavender:             "lavender",
	tcell.ColorLavenderBlush:        "lavenderblush",
	tcell.ColorLawnGreen:            "lawngreen",
	tcell.ColorLemonChiffon:         "lemonchiffon",
	tcell.ColorLightBlue:            "lightblue",
	tcell.ColorLightCoral:           "lightcoral",
	tcell.ColorLightCyan:            "lightcyan",
	tcell.ColorLightGoldenrodYellow: "lightgoldenrodyellow",
	tcell.ColorLightGray:            "lightgray",
	tcell.ColorLightGreen:           "lightgreen",
	tcell.ColorLightPink:            "lightpink",
	tcell.ColorLightSalmon:          "lightsalmon",
	tcell.ColorLightSeaGreen:        "lightseagreen",
	tcell.ColorLightSkyBlue:         "lightskyblue",
	tcell.ColorLightSlateGray:       "lightslategray",
	tcell.ColorLightSteelBlue:       "lightsteelblue",
	tcell.ColorLightYellow:          "lightyellow",
	tcell.ColorLimeGreen:            "limegreen",
	tcell.ColorLinen:                "linen",
	tcell.ColorMediumAquamarine:     "mediumaquamarine",
	tcell.ColorMediumBlue:           "mediumblue",
	tcell.ColorMediumOrchid:         "mediumorchid",
	tcell.ColorMediumPurple:         "mediumpurple",
	tcell.ColorMediumSeaGreen:       "mediumseagreen",
	tcell.ColorMediumSlateBlue:      "mediumslateblue",
	tcell.ColorMediumSpringGreen:    "mediumspringgreen",
	tcell.ColorMediumTurquoise:      "mediumturquoise",
	tcell.ColorMediumVioletRed:      "mediumvioletred",
	tcell.ColorMidnightBlue:         "midnightblue",
	tcell.ColorMintCream:            "mintcream",
	tcell.ColorMistyRose:            "mistyrose",
	tcell.ColorMoccasin:             "moccasin",
	tcell.ColorNavajoWhite:          "navajowhite",
	tcell.ColorOldLace:              "oldlace",
	tcell.ColorOliveDrab:            "olivedrab",
	tcell.ColorOrange:               "orange",
	tcell.ColorOrangeRed:            "orangered",
	tcell.ColorOrchid:               "orchid",
	tcell.ColorPaleGoldenrod:        "palegoldenrod",
	tcell.ColorPaleGreen:            "palegreen",
	tcell.ColorPaleTurquoise:        "paleturquoise",
	tcell.ColorPaleVioletRed:        "palevioletred",
	tcell.ColorPapayaWhip:           "papayawhip",
	tcell.ColorPeachPuff:            "peachpuff",
	tcell.ColorPeru:                 "peru",
	tcell.ColorPink:                 "pink",
	tcell.ColorPlum:                 "plum",
	tcell.ColorPowderBlue:           "powderblue",
	tcell.ColorRebeccaPurple:        "rebeccapurple",
	tcell.ColorRosyBrown:            "rosybrown",
	tcell.ColorRoyalBlue:            "royalblue",
	tcell.ColorSaddleBrown:          "saddlebrown",
	tcell.ColorSalmon:               "salmon",
	tcell.ColorSandyBrown:           "sandybrown",
	tcell.ColorSeaGreen:             "seagreen",
	tcell.ColorSeashell:             "seashell",
	tcell.ColorSienna:               "sienna",
	tcell.ColorSkyblue:              "skyblue",
	tcell.ColorSlateBlue:            "slateblue",
	tcell.ColorSlateGray:            "slategray",
	tcell.ColorSnow:                 "snow",
	tcell.ColorSpringGreen:          "springgreen",
	tcell.ColorSteelBlue:            "steelblue",
	tcell.ColorTan:                  "tan",
	tcell.ColorThistle:              "thistle",
	tcell.ColorTomato:               "tomato",
	tcell.ColorTurquoise:            "turquoise",
	tcell.ColorViolet:               "violet",
	tcell.ColorWheat:                "wheat",
	tcell.ColorWhiteSmoke:           "whitesmoke",
	tcell.ColorYellowGreen:          "yellowgreen",
	// Duplicate keys
	/*
	   tcell.ColorGray:                "grey",
	   tcell.ColorDimGray:             "dimgrey",
	   tcell.ColorDarkGray:            "darkgrey",
	   tcell.ColorDarkSlateGray:       "darkslategrey",
	   tcell.ColorLightGray:           "lightgrey",
	   tcell.ColorLightSlateGray:      "lightslategrey",
	   tcell.ColorSlateGray:           "slategrey",
	*/
}

// https://github.com/gdamore/tcell/blob/88b9c25c3c5ee48b611dfeca9a2e9cf07812c35e/color.go#L1021
func colorToString(color tcell.Color) string {
	if !color.Valid() {
		switch color {
		case tcell.ColorNone:
			return "none"
		case tcell.ColorDefault:
			return "default"
		case tcell.ColorReset:
			return "reset"
		}
		return ""
	}

	ret := colorToNamesMap[color]
	if ret == "" {
		return color.CSS()
	}
	return ret
}

func StyleToStyleTagString(style tcell.Style) string {
	foreground, background, attributeMask := style.Decompose()
	// https://pkg.go.dev/github.com/gdamore/tcell/v2#AttrMask
	attributeStyleTagNameLookup := []byte{'b', 'l', 'r', 'u', 'd', 'i', 's'}

	var attributesStringBuilder strings.Builder
	if attributeMask != 0 {
		for i := 0; i < len(attributeStyleTagNameLookup); i++ {
			if attributeMask&(1<<i) != 0 {
				attributesStringBuilder.WriteByte(attributeStyleTagNameLookup[i])
			}
		}
	}

	attributesString := attributesStringBuilder.String()
	if attributesString == "" {
		return "[" + colorToString(foreground) + ":" + colorToString(background) + "]"
	}

	return "[" + colorToString(foreground) + ":" + colorToString(background) + ":" + attributesString + "]"
}

// These file extensions have to be lowercase because we match for that
var imageTypes = []string{
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

var videoTypes = []string{
	".mp4",
	".webm",
	".mkv",
	".mov",
	".avi",
	".flv",
}

var audioTypes = []string{
	".wav",
	".flac",
	".mp3",
	".ogg",
	".m4a",
}

var archiveTypes = []string{
	".zip",
	".jar",
	".kra",
	".rar",

	// https://en.wikipedia.org/wiki/Tar_(computing)
	".tar.bz2", ".tb2", ".tbz", ".tbz2", ".tz2",
	".tar.gz", ".taz", ".tgz",
	".tar.lz",
	".tar.lzma", ".tlz",
	".tar.lzo",
	".tar.xz", ".tz", ".taz",
	".tar.zst", ".tzst",
}

var codeTypes = []string{
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
	".ps1",
	".bat",
	".vb",
	".vbs",
	".vbscript",
	".odin",
}

var documentTypes = []string{
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

var windowsExecutableTypes = []string{
	".exe",
	".msi",
}

// stat should be from an os.Lstat(). If stat is nil, it returns tcell.StyleDefault
func FileColor(stat os.FileInfo, path string) tcell.Style {
	if stat == nil {
		return tcell.StyleDefault
	}

	hasSuffixFromList := func(str string, list []string) bool {
		for _, e := range list {
			if strcase.HasSuffix(str, e) {
				return true
			}
		}

		return false
	}

	var ret tcell.Style

	if stat.IsDir() {
		return ret.Foreground(tcell.ColorBlue).Bold(true)
	} else if stat.Mode().IsRegular() {
		if stat.Mode()&0111 != 0 || (runtime.GOOS == "windows" && hasSuffixFromList(stat.Name(), windowsExecutableTypes)) { // Executable file
			return ret.Foreground(tcell.NewRGBColor(0, 255, 0)).Bold(true) // Green
		}
	} else if stat.Mode()&os.ModeSymlink != 0 {
		targetStat, err := os.Stat(path)
		if err == nil && targetStat.IsDir() {
			return ret.Foreground(tcell.ColorTeal).Bold(true)
		}

		return ret.Foreground(tcell.ColorTeal)
	} else {
		// Should not happen?
		return ret.Foreground(tcell.ColorDarkGray)
	}

	// In order of most hit: None (4420), Document (3422), Code (2049), Image (1257), Video (862), Archive (423), Audio (334)

	if hasSuffixFromList(stat.Name(), documentTypes) {
		return ret.Foreground(tcell.ColorGray)
	}

	if hasSuffixFromList(stat.Name(), codeTypes) {
		return ret.Foreground(tcell.ColorNavy)
	}

	if hasSuffixFromList(stat.Name(), imageTypes) {
		return ret.Foreground(tcell.ColorOlive)
	}

	if hasSuffixFromList(stat.Name(), videoTypes) {
		return ret.Foreground(tcell.ColorHotPink)
	}

	if hasSuffixFromList(stat.Name(), archiveTypes) {
		return ret.Foreground(tcell.ColorRed)
	}

	if hasSuffixFromList(stat.Name(), audioTypes) {
		return ret.Foreground(tcell.ColorPurple)
	}

	return ret.Foreground(tcell.ColorDefault)
}

func PathMatchesList(path string, matchList []string) bool {
	for _, match := range matchList {
		matched, _ := filepath.Match(match, filepath.Base(path))
		if matched {
			return true
		}
	}
	return false
}

func PathMatchesListCaseInsensitive(path string, matchList []string) bool {
	for _, match := range matchList {
		matched, _ := filepath.Match(strings.ToLower(match), strings.ToLower(filepath.Base(path)))
		if matched {
			return true
		}
	}
	return false
}

// We could maybe cache this to a certain extent
func ProgramsAndDescriptionsForFile(fen *Fen) ([]string, []string) {
	var programs []string
	var descriptions []string
	for _, programMatch := range fen.config.Open {
		matched := PathMatchesList(fen.sel, programMatch.Match) && !PathMatchesList(fen.sel, programMatch.DoNotMatch)

		if matched {
			if programMatch.Script != "" {
				programs = append(programs, programMatch.Script)
				descriptions = append(descriptions, "(Lua) User config")
			}
			for _, program := range programMatch.Program {
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
		programs = append(programs, filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")+" "+"url.dll,FileProtocolHandler")
		descriptions = append(descriptions, "Windows")
	} else {
		programs = append(programs, "xdg-open")
		descriptions = append(descriptions, "Linux/FreeBSD")
	}

	editor := os.Getenv("EDITOR")
	if editor != "" {
		programs = append(programs, editor)
		descriptions = append(descriptions, "$EDITOR")
	}
	programs = append(programs, "vim -p", "vi", "nano")
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

	if len(programs) != len(descriptions) {
		panic("In ProgramsAndDescriptionsForFile(): Length of programs and descriptions weren't the same")
	}

	return programs, descriptions
}

type FenOpenWithLuaGlobal struct {
	SelectedFiles []string
	ConfigPath    string
	Version       string
	RuntimeOS     string
}

// Returns the keys of theMap, order not guaranteed
func MapStringBoolKeys(theMap map[string]bool) []string {
	ret := make([]string, len(theMap))
	i := 0
	for k := range theMap {
		ret[i] = k
		i++
	}
	return ret
}

func OpenFile(fen *Fen, app *tview.Application, openWith string) error {
	if fen.config.PrintPathOnOpen {
		app.Stop()

		if len(fen.selected) <= 0 {
			if strings.ContainsRune(fen.sel, '\n') {
				fmt.Fprintln(os.Stderr, "The file you've selected has a newline (0x0a) in it's filename, exiting...")
				return nil
			}
			fmt.Println(fen.sel)
		} else {
			for selectedFile := range fen.selected {
				if strings.ContainsRune(selectedFile, '\n') {
					fmt.Fprintln(os.Stderr, "A file you've selected has a newline (0x0a) in it's filename, exiting...")
					return nil
				}
			}
			for selectedFile := range fen.selected {
				fmt.Println(selectedFile)
			}
		}
		return nil
	}

	if fen.config.NoWrite {
		return errors.New("Can't open files in no-write mode")
	}

	programsAndFallbacks, descriptions := ProgramsAndDescriptionsForFile(fen)
	if openWith != "" {
		programsAndFallbacks = append([]string{openWith}, programsAndFallbacks...)
		descriptions = append([]string{""}, descriptions...) // We need a description for any program, checked in the if below
	}

	if len(programsAndFallbacks) != len(descriptions) {
		panic("In OpenFile(): Length of programs and descriptions weren't the same")
	}

	app.Suspend(func() {
		for i, programOrScript := range programsAndFallbacks {
			description := descriptions[i]
			if description == "(Lua) User config" { // Hacky
				L := lua.NewState()
				defer L.Close()

				fenOpenWithLuaGlobal := &FenOpenWithLuaGlobal{
					ConfigPath: PathWithEndSeparator(filepath.Dir(fen.configFilePath)),
					Version:    version,
					RuntimeOS:  runtime.GOOS,
				}

				if len(fen.selected) > 0 {
					fenOpenWithLuaGlobal.SelectedFiles = MapStringBoolKeys(fen.selected)
				} else {
					fenOpenWithLuaGlobal.SelectedFiles = []string{fen.sel}
				}

				L.SetGlobal("fen", luar.New(L, fenOpenWithLuaGlobal))

				err := L.DoFile(programOrScript)
				if err != nil {
					fmt.Println("Lua error:")
					fmt.Println(err)
					PressAnyKeyToContinue("\x1b[1;30m"+pressAnyKeyToContinueText, "\x1b[1;30m"+pressEnterToContinueText)
					return
				}

				break
			}

			programSplitSpace := strings.Split(programOrScript, " ")

			programName := programSplitSpace[0]
			programArguments := []string{}
			if len(programSplitSpace) > 1 {
				programArguments = programSplitSpace[1:]
			}

			var cmd *exec.Cmd
			if len(fen.selected) <= 0 {
				cmd = exec.Command(programName, append(programArguments, fen.sel)...)
			} else {
				// FIXME: It would be nice if we kept track of the order of selected files,
				// or sort them first to make the argument order deterministic.
				cmd = exec.Command(programName, append(programArguments, MapStringBoolKeys(fen.selected)...)...)
			}
			cmd.Dir = fen.wd
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err == nil {
				break
			}
		}

		if fen.config.PauseOnOpenFile {
			PressAnyKeyToContinue("\x1b[1;30m"+"Press \x1b[4many key\x1b[0m\x1b[1;30m to continue...", "\x1b[1;30m"+pressEnterToContinueText)
		}
	})

	return nil
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

	if strings.HasSuffix(path, string(os.PathSeparator)) {
		panic("FilePathUniqueNameIfAlreadyExists got a file path ending in " + string(os.PathSeparator))
	}

	newPath := path
	for i := -1; ; i++ {
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

// Looping over the entire invisibleRunes array in isInvisible() showed up pretty high when profiling with pprof, so we generate a list of (start,end) ranges to check instead
// {9,13, 32,32, ..., 6155 6158, ..., 917760 917999}
func getInvisibleRunesAsRanges() []uint64 {
	var ret []uint64
	var lastRuneCode uint64

	// Invisible or blank runes like space
	// From https://github.com/flopp/invisible-characters/blob/main/main.go#L63 (Downloaded 2024-06-11 16:52 norway time, commit 0f7fc8a)
	var invisibleRunes = [...]uint64{9, 10, 11, 12, 13, 32, 127, 160, 173, 847, 1564, 4447, 4448, 6068, 6069, 6155, 6156, 6157, 6158, 7355, 7356, 8192, 8193, 8194, 8195, 8196, 8197, 8198, 8199, 8200, 8201, 8202, 8203, 8204, 8205, 8206, 8207, 8234, 8235, 8236, 8237, 8238, 8239, 8287, 8288, 8289, 8290, 8291, 8292, 8293, 8294, 8295, 8296, 8297, 8298, 8299, 8300, 8301, 8302, 8303, 10240, 12288, 12644, 65024, 65025, 65026, 65027, 65028, 65029, 65030, 65031, 65032, 65033, 65034, 65035, 65036, 65037, 65038, 65039, 65279, 65440, 65520, 65521, 65522, 65523, 65524, 65525, 65526, 65527, 65528, 65532, 78844, 119155, 119156, 119157, 119158, 119159, 119160, 119161, 119162, 917504, 917505, 917506, 917507, 917508, 917509, 917510, 917511, 917512, 917513, 917514, 917515, 917516, 917517, 917518, 917519, 917520, 917521, 917522, 917523, 917524, 917525, 917526, 917527, 917528, 917529, 917530, 917531, 917532, 917533, 917534, 917535, 917536, 917537, 917538, 917539, 917540, 917541, 917542, 917543, 917544, 917545, 917546, 917547, 917548, 917549, 917550, 917551, 917552, 917553, 917554, 917555, 917556, 917557, 917558, 917559, 917560, 917561, 917562, 917563, 917564, 917565, 917566, 917567, 917568, 917569, 917570, 917571, 917572, 917573, 917574, 917575, 917576, 917577, 917578, 917579, 917580, 917581, 917582, 917583, 917584, 917585, 917586, 917587, 917588, 917589, 917590, 917591, 917592, 917593, 917594, 917595, 917596, 917597, 917598, 917599, 917600, 917601, 917602, 917603, 917604, 917605, 917606, 917607, 917608, 917609, 917610, 917611, 917612, 917613, 917614, 917615, 917616, 917617, 917618, 917619, 917620, 917621, 917622, 917623, 917624, 917625, 917626, 917627, 917628, 917629, 917630, 917631, 917760, 917761, 917762, 917763, 917764, 917765, 917766, 917767, 917768, 917769, 917770, 917771, 917772, 917773, 917774, 917775, 917776, 917777, 917778, 917779, 917780, 917781, 917782, 917783, 917784, 917785, 917786, 917787, 917788, 917789, 917790, 917791, 917792, 917793, 917794, 917795, 917796, 917797, 917798, 917799, 917800, 917801, 917802, 917803, 917804, 917805, 917806, 917807, 917808, 917809, 917810, 917811, 917812, 917813, 917814, 917815, 917816, 917817, 917818, 917819, 917820, 917821, 917822, 917823, 917824, 917825, 917826, 917827, 917828, 917829, 917830, 917831, 917832, 917833, 917834, 917835, 917836, 917837, 917838, 917839, 917840, 917841, 917842, 917843, 917844, 917845, 917846, 917847, 917848, 917849, 917850, 917851, 917852, 917853, 917854, 917855, 917856, 917857, 917858, 917859, 917860, 917861, 917862, 917863, 917864, 917865, 917866, 917867, 917868, 917869, 917870, 917871, 917872, 917873, 917874, 917875, 917876, 917877, 917878, 917879, 917880, 917881, 917882, 917883, 917884, 917885, 917886, 917887, 917888, 917889, 917890, 917891, 917892, 917893, 917894, 917895, 917896, 917897, 917898, 917899, 917900, 917901, 917902, 917903, 917904, 917905, 917906, 917907, 917908, 917909, 917910, 917911, 917912, 917913, 917914, 917915, 917916, 917917, 917918, 917919, 917920, 917921, 917922, 917923, 917924, 917925, 917926, 917927, 917928, 917929, 917930, 917931, 917932, 917933, 917934, 917935, 917936, 917937, 917938, 917939, 917940, 917941, 917942, 917943, 917944, 917945, 917946, 917947, 917948, 917949, 917950, 917951, 917952, 917953, 917954, 917955, 917956, 917957, 917958, 917959, 917960, 917961, 917962, 917963, 917964, 917965, 917966, 917967, 917968, 917969, 917970, 917971, 917972, 917973, 917974, 917975, 917976, 917977, 917978, 917979, 917980, 917981, 917982, 917983, 917984, 917985, 917986, 917987, 917988, 917989, 917990, 917991, 917992, 917993, 917994, 917995, 917996, 917997, 917998, 917999}

	var start uint64
	length := uint64(1)
	for i, e := range invisibleRunes {
		if i == 0 {
			start = e
			lastRuneCode = e
			continue
		}

		if i == len(invisibleRunes)-1 {
			length++
		}

		if e-1 != lastRuneCode || i == len(invisibleRunes)-1 {
			ret = append(ret, start)
			ret = append(ret, start+length-1)
			length = 1
			start = e
		} else {
			length++
		}

		lastRuneCode = e
	}

	if len(ret)%2 != 0 {
		panic("getInvisibleRunesAsRanges failed to produce an even-numbered list")
	}

	return ret
}

var InvisibleRunesRanges = getInvisibleRunesAsRanges()

func isInvisible(c rune) bool {
	if !unicode.IsPrint(c) {
		return true
	}

	for i := 0; i < len(InvisibleRunesRanges); i += 2 {
		if c >= rune(InvisibleRunesRanges[i]) && c <= rune(InvisibleRunesRanges[i+1]) {
			return true
		}
	}

	return false
}

func RuneToPrintableCode(c rune) string {
	switch c {
	case '\a':
		return "\\a"
	case '\b':
		return "\\b"
	case '\f':
		return "\\f"
	case '\n':
		return "\\n"
	case '\r':
		return "\\r"
	case '\t':
		return "\\t"
	case '\v':
		return "\\v"
	}

	return "\\u" + strconv.FormatInt(int64(int32(c)), 16)
}

// Shows invisible runes as escape sequences like "\n", "\r", "\t", or "\u<HEX NUMBER>"
//
// Each of these will have a dark-red background, starting with "[:darkred]", and ending with "[-:-:-:-]"+defaultStyle which sets the colors back to whatever is specified in the defaultStyle style tag string
//
// Spaces are shown as normal, except leading and trailing ones, which will be shown as "\u20"
func FilenameInvisibleCharactersAsCodeHighlighted(filename, defaultStyle string) string {
	if filename == "" {
		panic("FilenameInvisibleCharactersAsCodeHighlighted got empty filename")
	}

	leadingInvisibleOrNonPrintableCharLength := 0
	for _, c := range filename {
		if isInvisible(c) {
			leadingInvisibleOrNonPrintableCharLength++
		} else {
			break
		}
	}

	trailingInvisibleOrNonPrintableCharLength := 0
	filenameRunes := []rune(filename)
	for i := len(filenameRunes) - 1; i >= leadingInvisibleOrNonPrintableCharLength; i-- {
		c := filenameRunes[i]
		if isInvisible(c) {
			trailingInvisibleOrNonPrintableCharLength++
		} else {
			break
		}
	}

	var ret strings.Builder
	for i, c := range filename {
		// Use printable codes for leading and trailing invisible or non-printable runes
		if i < leadingInvisibleOrNonPrintableCharLength || len(filename)-i <= trailingInvisibleOrNonPrintableCharLength {
			ret.WriteString("[:darkred]" + RuneToPrintableCode(c) + "[-:-:-:-]" + defaultStyle)
			continue
		}

		// For the rest, don't use printable codes for space
		if c != ' ' && isInvisible(c) {
			ret.WriteString("[:darkred]" + RuneToPrintableCode(c) + "[-:-:-:-]" + defaultStyle)
			continue
		}

		ret.WriteRune(c)
	}
	return ret.String()
}

// Returns the length printed
func PrintFilenameInvisibleCharactersAsCodeHighlighted(screen tcell.Screen, x, y, maxWidth int, filename string, style tcell.Style) int {
	if filename == "" {
		panic("PrintFilenameInvisibleCharactersAsCodeHighlighted got empty filename")
	}

	leadingInvisibleOrNonPrintableCharLength := 0
	for _, c := range filename {
		if isInvisible(c) {
			leadingInvisibleOrNonPrintableCharLength++
		} else {
			break
		}
	}

	trailingInvisibleOrNonPrintableCharLength := 0
	filenameRunes := []rune(filename)
	for i := len(filenameRunes) - 1; i >= leadingInvisibleOrNonPrintableCharLength; i-- {
		c := filenameRunes[i]
		if isInvisible(c) {
			trailingInvisibleOrNonPrintableCharLength++
		} else {
			break
		}
	}

	offset := 0
	for i, c := range filename {
		if offset >= maxWidth-2 {
			screen.SetContent(x+offset, y, missingSpaceRune, nil, style)
			offset++
			return offset
		}

		// Use printable codes for leading and trailing invisible or non-printable runes
		if i < leadingInvisibleOrNonPrintableCharLength || len(filename)-i <= trailingInvisibleOrNonPrintableCharLength {
			printableCode := RuneToPrintableCode(c)
			for _, character := range printableCode {
				if offset >= maxWidth-2 {
					screen.SetContent(x+offset, y, missingSpaceRune, nil, style)
					offset++
					return offset
				}

				screen.SetContent(x+offset, y, character, nil, tcell.StyleDefault.Background(tcell.ColorDarkRed))
				offset++
			}

			continue
		}

		// For the rest, don't use printable codes for space
		if c != ' ' && isInvisible(c) {
			printableCode := RuneToPrintableCode(c)
			for _, character := range printableCode {
				if offset >= maxWidth-2 {
					screen.SetContent(x+offset, y, missingSpaceRune, nil, style)
					offset++
					return offset
				}

				screen.SetContent(x+offset, y, character, nil, tcell.StyleDefault.Background(tcell.ColorDarkRed))
				offset++
			}
			continue
		}

		screen.SetContent(x+offset, y, c, nil, style)
		offset++
	}

	return offset
}

// Falls back to pressEnterText if an error occurred.
// NOTE: Since this enables terminal raw mode, you need an explicit carriage return for every newline in the arguments (atleast on xterm, Linux)
func PressAnyKeyToContinue(pressAnyKeyText, pressEnterText string) {
	defer fmt.Print("\x1b[0m\n\n")

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Print(pressEnterText)
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	fmt.Print(pressAnyKeyText)
	b := make([]byte, 1)
	os.Stdin.Read(b)

	term.Restore(int(os.Stdin.Fd()), oldState)
}

func GetShellArgs() [2]string {
	if runtime.GOOS == "windows" {
		return [2]string{"cmd", "/C"}
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return [2]string{shell, "-c"}
}

// Invokes system shell command
func InvokeShell(command, workingDirectory string) error {
	shellArgs := GetShellArgs()
	cmd := exec.Command(shellArgs[0], shellArgs[1], command)

	cmd.Dir = workingDirectory
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SetClipboardLinuxXClip(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	var b bytes.Buffer
	b.WriteString(text)
	cmd.Stdin = &b
	return cmd.Run()
}

func SHA256HashSum(path string) ([]byte, error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()

	data := make([]byte, stat.Size())
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, err
	}

	if _, err := hash.Write(data); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// Returns nil error if the string slice has a duplicate value, returns first duplicate value found
func StringSliceHasDuplicate(strSlice []string) (string, error) {
	valuesMap := make(map[string]bool)
	for _, value := range strSlice {
		_, duplicate := valuesMap[value]
		if duplicate {
			return value, nil
		}

		valuesMap[value] = true
	}

	return "", errors.New("No duplicate found")
}

// Returns a random string of length numCharacters containing lowercase a-z and 0-9.
func RandomStringPathSafe(numCharacters int) string {
	if numCharacters < 0 {
		numCharacters = 0
	}

	// It is important not to use mixed case characters because some filesystems are case-insensitive
	alphabet := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, numCharacters)

	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func IsYes(str string) bool {
	return strings.ToLower(strings.TrimSpace(str)) == "y"
}

func CurrentWorkingDirectory() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		if runtime.GOOS == "windows" || runtime.GOOS == "plan9" {
			return "", errors.New("unable to determine current working directory")
		}

		path = os.Getenv("PWD")
		if path == "" {
			return "", errors.New("PWD environment variable empty")
		}
	}

	return path, nil
}

// Splits an absolute path by os.PathSeparator into its components
// For the input "C:\Users\user\Desktop\file.txt", it will return:
// ["C:\", "C:\Users", "C:\Users\user", "C:\Users\user\Desktop", "C:\Users\user\Desktop\file.txt"]
func SplitPath(path string) []string {
	if !filepath.IsAbs(path) {
		panic("SplitPath() was passed a non-absolute path")
	}

	return splitPathTestable(path, os.PathSeparator)
}

func splitPathTestable(path string, pathSeparator rune) []string {
	split := strings.Split(path, string(pathSeparator))
	split = slices.DeleteFunc(split, func(x string) bool {
		return x == ""
	})

	if pathSeparator == '/' {
		if path != "" {
			split = append([]string{"/"}, split...)
		}
	} else if pathSeparator == '\\' {
		if len(split) > 0 {
			split[0] += "\\"
		}
	} else {
		panic("Unsupported OS path separator")
	}

	ret := make([]string, len(split))

	var pathConcat strings.Builder
	for i, pathSplit := range split {
		pathConcat.WriteString(pathSplit)

		ret[i] = pathConcat.String()

		// We should technically be indexing by runes, but we only support "/" and "\\", so it's fine.
		if pathSplit[len(pathSplit)-1] != byte(pathSeparator) {
			pathConcat.WriteRune(pathSeparator)
		}
	}

	return ret
}

// Panics on empty string!
// Returns the number prefix of a string
// e.g. "123hello" -> "123"
// and  "hello" -> ""
func NumberPrefix(s string) string {
	if len(s) == 0 {
		panic("NumberPrefix() was passed an empty string")
	}

	count := 0
	for _, char := range s {
		if char < '0' || char > '9' {
			break
		}

		count++
	}

	return s[:count]
}

// Compares two positive numerical strings.
// Returns  0 if num1 == num2
// Returns  1 if num1 > num2
// Returns -1 if num1 < num2
func CompareNumericalStrings(num1, num2 string) int {
	num1Strip := strings.TrimLeft(num1, "0")
	num2Strip := strings.TrimLeft(num2, "0")
	len1 := len(num1Strip)
	len2 := len(num2Strip)

	if len1 > len2 {
		return 1
	}

	if len1 < len2 {
		return -1
	}

	for i := 0; i < len1; i++ {
		if num1Strip[i] > num2Strip[i] {
			return 1
		} else if num1Strip[i] < num2Strip[i] {
			return -1
		}
	}

	return 0 // Both numbers are equal
}

// Does tilde expansion on non-windows operating systems
// If it can't find the home directory, it will return the input unchanged.
func ExpandTilde(path string) string {
	if runtime.GOOS == "windows" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return expandTildeTestable(path, home)
}

func expandTildeTestable(path string, homeDir string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[1:])
	}

	return path
}

// The valid values for the caseSensitivity parameter are defined in ValidFilenameSearchCaseValues (fen.go)
func FindSubstringAllStartIndices(s, searchText, caseSensitivity string) []int {
	if s == "" || searchText == "" {
		return []int{}
	}

	var result []int

	var stringIndexFunc func(s, substr string) int
	if caseSensitivity == CASE_INSENSITIVE {
		stringIndexFunc = strcase.Index
	} else if caseSensitivity == CASE_SENSITIVE {
		stringIndexFunc = strings.Index
	} else {
		panic("FindSubstringAllStartIndices(): Invalid fen.filename_search.Case value: " + caseSensitivity)
	}

	i := 0
	for limit := 0; limit < 100; limit += 1 { // Stop after 100 iterations
		if i >= len(s) {
			break
		}

		found := stringIndexFunc(s[i:], searchText)
		if found == -1 {
			break
		}

		i += found
		result = append(result, i)
		i += len(searchText)
	}

	return result
}

type Slice struct {
	start  int
	length int
}

func SpreadArrayIntoSlicesForGoroutines(arrayLength, numGoroutines int) []Slice {
	if arrayLength == 0 {
		return []Slice{}
	}

	if numGoroutines <= 1 {
		return []Slice{
			{0, arrayLength},
		}
	}

	// More goroutines than there are elements, use arrayLength goroutines instead.
	// That is, 1 goroutine per element...
	if numGoroutines >= arrayLength {
		var result []Slice
		for i := 0; i < arrayLength; i++ {
			result = append(result, Slice{i, 1})
		}
		return result
	}

	var result []Slice
	lengthPerGoroutine := arrayLength / numGoroutines

	rollingIndex := 0
	for i := 0; i < numGoroutines-1; i++ {
		result = append(result, Slice{
			start:  rollingIndex,
			length: lengthPerGoroutine,
		})

		rollingIndex += lengthPerGoroutine
	}

	// Last goroutine will handle the last part of the array
	result = append(result, Slice{
		start:  rollingIndex,
		length: arrayLength - rollingIndex,
	})

	return result
}
