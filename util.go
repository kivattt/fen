package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
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

func EntrySizeText(entryInfo fs.FileInfo, path string, hiddenFiles bool) (string, error) {
	if !entryInfo.IsDir() {
		return BytesToHumanReadableUnitString(uint64(entryInfo.Size()), 2), nil
	} else {
		count, err := FolderFileCount(path, hiddenFiles)
		if err != nil {
			return "", err
		}

		return strconv.Itoa(count), nil
	}
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

func PathMatchesList(path string, matchList []string) bool {
	for _, match := range matchList {
		matched, _ := filepath.Match(match, filepath.Base(path))
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

func OpenFile(fen *Fen, app *tview.Application, openWith string) {
	if fen.config.PrintPathOnOpen && openWith == "" {
		app.Stop()
		if len(fen.selected) <= 0 {
			if strings.ContainsRune(fen.sel, '\n') {
				fmt.Fprintln(os.Stderr, "The file you've selected has a newline (0x0a) in it's filename, exiting...")
				return
			}
			fmt.Println(fen.sel)
		} else {
			for _, selectedFile := range fen.selected {
				if strings.ContainsRune(selectedFile, '\n') {
					fmt.Fprintln(os.Stderr, "A file you've selected has a newline (0x0a) in it's filename, exiting...")
					return
				}
			}
			fmt.Println(strings.Join(fen.selected, "\n"))
		}
		return
	}

	if fen.config.NoWrite {
		fen.bottomBar.TemporarilyShowTextInstead("Can't open files in no-write mode")
		return
	}

	programsAndFallbacks, descriptions := ProgramsAndDescriptionsForFile(fen)
	if openWith != "" {
		programsAndFallbacks = append([]string{openWith}, programsAndFallbacks...)
		descriptions = append([]string{"hi"}, descriptions...)
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

				userConfigDir, _ := os.UserConfigDir()
				fenOpenWithLuaGlobal := &FenOpenWithLuaGlobal{
					ConfigPath: PathWithEndSeparator(filepath.Join(userConfigDir, "fen")),
					Version:    version,
					RuntimeOS:  runtime.GOOS,
				}

				if len(fen.selected) > 0 {
					fenOpenWithLuaGlobal.SelectedFiles = fen.selected
				} else {
					fenOpenWithLuaGlobal.SelectedFiles = []string{fen.sel}
				}

				L.SetGlobal("fen", luar.New(L, fenOpenWithLuaGlobal))

				err := L.DoFile(programOrScript)
				if err != nil {
					fmt.Println("Lua error:")
					fmt.Println(err)
					fmt.Println("Press Enter to continue...")
					bufio.NewReader(os.Stdin).ReadString('\n')
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
				cmd = exec.Command(programName, append(programArguments, fen.selected...)...)
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

// Invisible or blank runes like space
// From https://github.com/flopp/invisible-characters/blob/main/main.go#L63 (Downloaded 2024-06-11 16:52 norway time, commit 0f7fc8a)
var invisibleRunes = [...]uint64{9, 10, 11, 12, 13, 32, 127, 160, 173, 847, 1564, 4447, 4448, 6068, 6069, 6155, 6156, 6157, 6158, 7355, 7356, 8192, 8193, 8194, 8195, 8196, 8197, 8198, 8199, 8200, 8201, 8202, 8203, 8204, 8205, 8206, 8207, 8234, 8235, 8236, 8237, 8238, 8239, 8287, 8288, 8289, 8290, 8291, 8292, 8293, 8294, 8295, 8296, 8297, 8298, 8299, 8300, 8301, 8302, 8303, 10240, 12288, 12644, 65024, 65025, 65026, 65027, 65028, 65029, 65030, 65031, 65032, 65033, 65034, 65035, 65036, 65037, 65038, 65039, 65279, 65440, 65520, 65521, 65522, 65523, 65524, 65525, 65526, 65527, 65528, 65532, 78844, 119155, 119156, 119157, 119158, 119159, 119160, 119161, 119162, 917504, 917505, 917506, 917507, 917508, 917509, 917510, 917511, 917512, 917513, 917514, 917515, 917516, 917517, 917518, 917519, 917520, 917521, 917522, 917523, 917524, 917525, 917526, 917527, 917528, 917529, 917530, 917531, 917532, 917533, 917534, 917535, 917536, 917537, 917538, 917539, 917540, 917541, 917542, 917543, 917544, 917545, 917546, 917547, 917548, 917549, 917550, 917551, 917552, 917553, 917554, 917555, 917556, 917557, 917558, 917559, 917560, 917561, 917562, 917563, 917564, 917565, 917566, 917567, 917568, 917569, 917570, 917571, 917572, 917573, 917574, 917575, 917576, 917577, 917578, 917579, 917580, 917581, 917582, 917583, 917584, 917585, 917586, 917587, 917588, 917589, 917590, 917591, 917592, 917593, 917594, 917595, 917596, 917597, 917598, 917599, 917600, 917601, 917602, 917603, 917604, 917605, 917606, 917607, 917608, 917609, 917610, 917611, 917612, 917613, 917614, 917615, 917616, 917617, 917618, 917619, 917620, 917621, 917622, 917623, 917624, 917625, 917626, 917627, 917628, 917629, 917630, 917631, 917760, 917761, 917762, 917763, 917764, 917765, 917766, 917767, 917768, 917769, 917770, 917771, 917772, 917773, 917774, 917775, 917776, 917777, 917778, 917779, 917780, 917781, 917782, 917783, 917784, 917785, 917786, 917787, 917788, 917789, 917790, 917791, 917792, 917793, 917794, 917795, 917796, 917797, 917798, 917799, 917800, 917801, 917802, 917803, 917804, 917805, 917806, 917807, 917808, 917809, 917810, 917811, 917812, 917813, 917814, 917815, 917816, 917817, 917818, 917819, 917820, 917821, 917822, 917823, 917824, 917825, 917826, 917827, 917828, 917829, 917830, 917831, 917832, 917833, 917834, 917835, 917836, 917837, 917838, 917839, 917840, 917841, 917842, 917843, 917844, 917845, 917846, 917847, 917848, 917849, 917850, 917851, 917852, 917853, 917854, 917855, 917856, 917857, 917858, 917859, 917860, 917861, 917862, 917863, 917864, 917865, 917866, 917867, 917868, 917869, 917870, 917871, 917872, 917873, 917874, 917875, 917876, 917877, 917878, 917879, 917880, 917881, 917882, 917883, 917884, 917885, 917886, 917887, 917888, 917889, 917890, 917891, 917892, 917893, 917894, 917895, 917896, 917897, 917898, 917899, 917900, 917901, 917902, 917903, 917904, 917905, 917906, 917907, 917908, 917909, 917910, 917911, 917912, 917913, 917914, 917915, 917916, 917917, 917918, 917919, 917920, 917921, 917922, 917923, 917924, 917925, 917926, 917927, 917928, 917929, 917930, 917931, 917932, 917933, 917934, 917935, 917936, 917937, 917938, 917939, 917940, 917941, 917942, 917943, 917944, 917945, 917946, 917947, 917948, 917949, 917950, 917951, 917952, 917953, 917954, 917955, 917956, 917957, 917958, 917959, 917960, 917961, 917962, 917963, 917964, 917965, 917966, 917967, 917968, 917969, 917970, 917971, 917972, 917973, 917974, 917975, 917976, 917977, 917978, 917979, 917980, 917981, 917982, 917983, 917984, 917985, 917986, 917987, 917988, 917989, 917990, 917991, 917992, 917993, 917994, 917995, 917996, 917997, 917998, 917999}

func isInvisible(c rune) bool {
	if !unicode.IsPrint(c) {
		return true
	}

	for _, invisible := range invisibleRunes {
		if c == rune(invisible) {
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

// Shows invisible runes as escape sequences like "\n", "\r", "\t", or "\u<NUMBER>"
//
// Each of these will have a dark-red background, starting with "[:darkred]", and ending with "[-:-:-:-]"+defaultStyle set the colors back to whatever is specified in the defaultStyle style tag string
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
	for i := len(filenameRunes) - 1; i >= 0; i-- {
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
