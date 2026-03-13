package gogitstatus

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	//"github.com/sabhiram/go-gitignore"
	ignore "github.com/botondmester/goignore"
)

// Debug output
var gogitstatus_debug_profiling = false
var gogitstatus_debug_ignored = false
var gogitstatus_debug_goroutine_slices = false

var gogitstatus_debug_disable_skipdir = false
var gogitstatus_debug_skipdir = false

// A small subset of a Git index entry
type GitIndexEntry struct {
	MetadataChangedTimeSeconds     uint32 // ctime
	MetadataChangedTimeNanoSeconds uint32 // ctime
	ModifiedTimeSeconds            uint32
	ModifiedTimeNanoSeconds        uint32
	Mode                           uint32   // Contains the file type and unix permission bits
	FileSize                       uint32   // Size of the file in bytes from stat(2), truncated to 32-bit
	Hash                           [20]byte // 20 bytes for the standard SHA-1
}

// This function is only used for path lengths in the .git/index longer than 0xffe bytes
// TODO: Can speed this up by first reading 0xfff bytes, and then 8 bytes at a time until the last byte of an 8-byte section is a null byte
func readIndexEntryPathName(reader *bytes.Reader) (strings.Builder, error) {
	var ret strings.Builder

	// Entry length so far
	entryLength := 40 + 20 + 2

	// FIXME: Try to do this on the stack instead
	singleByteSlice := make([]byte, 1)
	for {
		_, err := io.ReadFull(reader, singleByteSlice)
		if err != nil {
			return ret, errors.New("Invalid size, readIndexEntryPathName failed: " + err.Error())
		}

		b := singleByteSlice[0]

		if b == 0 {
			break
		} else {
			ret.WriteByte(b)
			entryLength++
		}
	}

	// Read up to 7 extra null padding bytes
	n := 8 - (entryLength % 8)
	if n == 0 {
		n = 8
	}
	n-- // We already read 1 null byte

	b := make([]byte, n)
	_, err := io.ReadFull(reader, b)
	if err != nil {
		return ret, errors.New("Invalid size, readIndexEntryPathName failed while seeking over null bytes: " + err.Error())
	}

	for _, e := range b {
		if e != 0 {
			return ret, errors.New("Non-null byte found in null padding of length " + strconv.Itoa(n))
		}
	}

	return ret, nil
}

// Returns the relative paths mapping to the GitIndexEntry
// Parses a Git Index file (version 2)
func ParseGitIndex(ctx context.Context, path string) (map[string]GitIndexEntry, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !stat.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// The mmap implementation is effectively the same speed as the plain io.ReadFull in this case
	// because we're only reading 1 file here. But let's use it when we can
	data, err := openFileData(file, stat)
	if err != nil {
		return nil, err
	}
	defer closeFileData(data)

	return ParseGitIndexFromMemory(ctx, data, -1)
}

// Returns the relative paths mapping to the GitIndexEntry
// Parses a Git Index file (version 2)
// Passing a negative value e.g. -1 to maxEntriesToPreAllocate means there will be no limit. Otherwise, we will only pre-allocate up to that many entries.
func ParseGitIndexFromMemory(ctx context.Context, data []byte, maxEntriesToPreAllocate int) (map[string]GitIndexEntry, error) {
	reader := bytes.NewReader(data)

	headerBytes := make([]byte, 12)
	_, err := io.ReadFull(reader, headerBytes)
	if err != nil {
		return nil, err
	}

	if !bytes.HasPrefix(headerBytes, []byte{'D', 'I', 'R', 'C'}) {
		return nil, errors.New("invalid header, missing \"DIRC\"")
	}

	version := binary.BigEndian.Uint32(headerBytes[4:8])
	if version != 2 {
		return nil, errors.New("unsupported version: " + strconv.FormatInt(int64(version), 10))
	}

	numEntries := binary.BigEndian.Uint32(headerBytes[8:12])
	var entries map[string]GitIndexEntry
	if maxEntriesToPreAllocate < 0 {
		entries = make(map[string]GitIndexEntry, numEntries)
	} else {
		entries = make(map[string]GitIndexEntry, min(uint32(maxEntriesToPreAllocate), numEntries))
	}

	flagsBytes := make([]byte, 2)         // 16 bits 'flags' field
	modeBytes := make([]byte, 4)          // 32 bits
	fileSizeBytes := make([]byte, 4)      // 32 bits
	eightBytes := make([]byte, 8)         // 64 bits
	hashBytes := make([]byte, 20)         // 160 bits
	pathNameBuffer := make([]byte, 0xffe) // We allocate enough for the largest possible known-size (not null-terminated) Git path name length.

	var entryIndex uint32
	for entryIndex = 0; entryIndex < numEntries; entryIndex++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Read 64-bit metadata changed time (ctime)
			if _, err := io.ReadFull(reader, eightBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 64-bit metadata changed time (ctime) within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			ctimeSeconds := binary.BigEndian.Uint32(eightBytes[:4])
			ctimeNanoSeconds := binary.BigEndian.Uint32(eightBytes[4:])

			// Read 64-bit modified time (mTime)
			if _, err := io.ReadFull(reader, eightBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 64-bit modified time within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			mTimeSeconds := binary.BigEndian.Uint32(eightBytes[:4])
			mTimeNanoSeconds := binary.BigEndian.Uint32(eightBytes[4:])

			// Seek to 32-bit mode
			if _, err := reader.Seek(8, 1); err != nil { // 64 bits
				return nil, errors.New("invalid size, unable to seek to 32-bit mode within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			// Read 32-bit mode
			if _, err := io.ReadFull(reader, modeBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 32-bit mode within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			mode := binary.BigEndian.Uint32(modeBytes)

			// Seek to 32-bit file size
			if _, err := reader.Seek(8, 1); err != nil { // 64 bits
				return nil, errors.New("invalid size, unable to seek to 32-bit file size within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			// Read 32-bit file size
			if _, err := io.ReadFull(reader, fileSizeBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 32-bit mode within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			fileSize := binary.BigEndian.Uint32(fileSizeBytes)

			// Read hash data
			if _, err := io.ReadFull(reader, hashBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 20-byte SHA-1 hash at index " + strconv.FormatUint(uint64(entryIndex), 10))
			}

			if _, err := io.ReadFull(reader, flagsBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 2-byte flags field at index " + strconv.FormatUint(uint64(entryIndex), 10))
			}

			flags := binary.BigEndian.Uint16(flagsBytes)
			nameLength := flags & 0xfff

			var pathName strings.Builder // TODO: Do we really want this to be a string builder? Might be faster to avoid it entirely?
			if nameLength == 0xfff {     // Path name length >= 0xfff, need to manually find null bytes
				// Read variable-length path name
				pathName, err = readIndexEntryPathName(reader)
				if err != nil {
					return nil, err
				}
			} else {
				if _, err := io.ReadFull(reader, pathNameBuffer[:nameLength]); err != nil {
					return nil, errors.New("invalid size, unable to read path name of size " + strconv.FormatUint(uint64(nameLength), 10) + " at index " + strconv.FormatUint(uint64(entryIndex), 10))
				}

				pathName.Write(pathNameBuffer[:nameLength])
				entryLength := 40 + 20 + 2 // Entry length so far
				// Read up to 8 null padding bytes
				n := 8 - ((int(nameLength) + entryLength) % 8)
				if n == 0 {
					n = 8
				}

				if _, err = io.ReadFull(reader, eightBytes[:n]); err != nil {
					return nil, errors.New("invalid size, unable to read path name null bytes of size " + strconv.FormatUint(uint64(n), 10) + " at index " + strconv.FormatUint(uint64(entryIndex), 10))
				}

				for _, e := range eightBytes[:n] {
					if e != 0 {
						return nil, errors.New("non-null byte found in null padding of length " + strconv.FormatUint(uint64(n), 10))
					}
				}
			}

			entries[pathName.String()] = GitIndexEntry{
				MetadataChangedTimeSeconds:     ctimeSeconds,
				MetadataChangedTimeNanoSeconds: ctimeNanoSeconds,
				ModifiedTimeSeconds:            mTimeSeconds,
				ModifiedTimeNanoSeconds:        mTimeNanoSeconds,
				Mode:                           mode,
				FileSize:                       fileSize,
				Hash:                           [20]byte(hashBytes),
			}
		}
	}

	return entries, nil
}

/*func convertLFToCRLF(data []byte) []byte {

}*/

func convertCRLFToLF(data []byte) []byte {
	out := make([]byte, 0, len(data))

	// Poor bytes.IndexByte() implementation
	// It's 2x faster in the average case, but 13x slower in worst case compared to the simple one below.

	/*i := bytes.IndexByte(data, '\r')
	if i == -1 {
		return data
	}
	out = append(out, data[:i]...)

	for {
		oldI := i
		next := bytes.IndexByte(data[i+1:], '\r')
		if next == -1 {
			i += next + 1
			out = append(out, data[oldI+1:]...)
			return out
		}

		i += next + 1
		out = append(out, data[oldI+1:i]...)
	}*/

	for _, c := range data {
		if c != '\r' {
			out = append(out, c)
		}
	}

	return out
}

func hashMatchesFileOrWithLineEndingConvertedHack(hash []byte, path string, stat os.FileInfo) bool {
	if hashMatchesFile(hash, path, stat) {
		return true
	}

	// HACK: If the hash doesn't match, we also attempt to hash with converted line endings (CRLF -> LF)
	// The proper way to go about this would be to figure out if/how to convert line endings via something like .gitattributes
	// I think there is no LF -> CRLF conversion. Only CRLF -> LF.

	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	data, err := openFileData(file, stat)
	if err != nil {
		return false
	}
	defer closeFileData(data)

	crlf := convertCRLFToLF(data)
	return hashMatches(hash, crlf)
}

func hashMatchesFile(hash []byte, path string, stat os.FileInfo) bool {
	// Symlinks are hashed with the target path, not the data of the target file
	// On Windows, symlinks are stored as regular files (with target path as the file data), so we handle them as such later
	if runtime.GOOS != "windows" && stat.Mode()&os.ModeSymlink != 0 /*|| !stat.Mode().IsRegular()*/ {
		targetPath, err := os.Readlink(path)
		if err != nil {
			return false
		}

		return hashMatches(hash, []byte(targetPath))
	}

	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	data, err := openFileData(file, stat)
	if err != nil {
		return false
	}
	defer closeFileData(data)

	return hashMatches(hash, data)
}

func hashMatches(hash, data []byte) bool {
	newHash := sha1.New()
	_, err := newHash.Write(append([]byte("blob "+strconv.FormatInt(int64(len(data)), 10)), 0))
	if err != nil {
		return false
	}

	_, err = newHash.Write(data)
	if err != nil {
		return false
	}

	return reflect.DeepEqual(hash, newHash.Sum(nil))
}

type WhatChanged uint8

const (
	// https://github.com/git/git/blob/ef8ce8f3d4344fd3af049c17eeba5cd20d98b69f/statinfo.h#L35
	MTIME_CHANGED WhatChanged = 0x0001 // We don't use this (yet)
	CTIME_CHANGED WhatChanged = 0x0002
	OWNER_CHANGED WhatChanged = 0x0004
	MODE_CHANGED  WhatChanged = 0x0008
	INODE_CHANGED WhatChanged = 0x0010 // We don't use this (yet)
	DATA_CHANGED  WhatChanged = 0x0020
	TYPE_CHANGED  WhatChanged = 0x0040

	DELETED = 0x0080
)

var whatChangedToStringMap = map[WhatChanged]string{
	MTIME_CHANGED: "MTIME_CHANGED",
	CTIME_CHANGED: "CTIME_CHANGED",
	OWNER_CHANGED: "OWNER_CHANGED",
	MODE_CHANGED:  "MODE_CHANGED",
	INODE_CHANGED: "INODE_CHANGED",
	DATA_CHANGED:  "DATA_CHANGED",
	TYPE_CHANGED:  "TYPE_CHANGED",

	DELETED: "DELETED",
}

var stringToWhatChangedMap = map[string]WhatChanged{
	"MTIME_CHANGED": MTIME_CHANGED,
	"CTIME_CHANGED": CTIME_CHANGED,
	"OWNER_CHANGED": OWNER_CHANGED,
	"MODE_CHANGED":  MODE_CHANGED,
	"INODE_CHANGED": INODE_CHANGED,
	"DATA_CHANGED":  DATA_CHANGED,
	"TYPE_CHANGED":  TYPE_CHANGED,

	"DELETED": DELETED,
}

func WhatChangedToString(whatChanged WhatChanged) string {
	var masksMatched []string

	for k, v := range whatChangedToStringMap {
		if whatChanged&k != 0 {
			masksMatched = append(masksMatched, v)
		}
	}

	return strings.Join(masksMatched, ",")
}

func StringToWhatChanged(text string) WhatChanged {
	split := strings.Split(text, ",")
	var ret WhatChanged
	for _, e := range split {
		ret |= stringToWhatChangedMap[e]
	}
	return ret
}

const OBJECT_TYPE_MASK = 0b1111 << 12

const REGULAR_FILE = 0b1000 << 12
const SYMBOLIC_LINK = 0b1010 << 12
const GITLINK = 0b1110 << 12

// Returns 0 if the file is unchanged.
// If you pass this a nil value for stat, it will return 0.
// https://github.com/git/git/blob/ef8ce8f3d4344fd3af049c17eeba5cd20d98b69f/read-cache.c#L307
func fileChanged(entry GitIndexEntry, entryFullPath string, stat os.FileInfo) WhatChanged {
	if stat == nil {
		return 0 // Deleted file
	}

	var whatChanged WhatChanged

	mTimeUnchanged := stat.ModTime() == time.Unix(int64(entry.ModifiedTimeSeconds), int64(entry.ModifiedTimeNanoSeconds))
	cTimeUnchanged := isCTimeUnchanged(stat, int64(entry.ModifiedTimeSeconds), int64(entry.MetadataChangedTimeNanoSeconds))

	// TODO: Use ctime to prevent hash-check, and mtime to prevent mode check? Look into Git source code for this
	if mTimeUnchanged && cTimeUnchanged {
		return 0
	}

	switch entry.Mode & OBJECT_TYPE_MASK {
	case REGULAR_FILE:
		if !stat.Mode().IsRegular() {
			whatChanged |= TYPE_CHANGED
		}

		// https://github.com/git/git/blob/ef8ce8f3d4344fd3af049c17eeba5cd20d98b69f/read-cache.c#L317
		// Windows only stores the mode permission bits in .git/index, not on disk
		if runtime.GOOS != "windows" && fs.FileMode(entry.Mode)&fs.ModePerm&0100 != stat.Mode()&fs.ModePerm&0100 {
			whatChanged |= MODE_CHANGED
		}
	case SYMBOLIC_LINK:
		// Symbolic links are stored as regular files on Windows
		if runtime.GOOS != "windows" && stat.Mode()&os.ModeSymlink == 0 /*&& !stat.Mode().IsRegular()*/ {
			whatChanged |= TYPE_CHANGED
		}
	case GITLINK:
		if !stat.IsDir() {
			whatChanged |= TYPE_CHANGED
		}
		return whatChanged
	default:
		panic("Unknown git index entry mode:" + strconv.FormatInt(int64(entry.Mode), 10))
	}

	// TODO: Store mtime and ctime to check for change here, as is done in the match_stat_data() function in Git

	if entry.FileSize != uint32(stat.Size()) {
		whatChanged |= DATA_CHANGED
	} else if !hashMatchesFileOrWithLineEndingConvertedHack(entry.Hash[:], entryFullPath, stat) {
		whatChanged |= DATA_CHANGED
	}

	return whatChanged
}

type ChangedFile struct {
	WhatChanged WhatChanged
	Untracked   bool // true = Untracked, false = Unstaged
}

func ignoreMatch(path string, ignoresMap map[string]*ignore.GitIgnore) bool {
	// TODO: Use strings.indexByte directly without temporary storage to speed up filepath.Dir()?

	dir := ""
	if path[len(path)-1] == '/' {
		// We hint something is a folder with a trailing '/', so remove it to get the real parent folder
		dir = filepath.Dir(path[:len(path)-1])
	} else {
		dir = filepath.Dir(path)
	}

	for {
		// Faster than filepath.Rel()
		var rel string
		if dir == "." {
			rel = path
		} else {
			rel = path[len(dir)+1:]
		}

		if rel == "" {
			panic("unreachable")
		}

		ignore, ok := ignoresMap[dir]
		if ok {
			// We don't need to do filepath.ToSlash(rel), since it's done inside goignore.
			if ignore.MatchesPath(rel) {
				return true
			}
		}

		// Debugging
		//fmt.Println("dir lookup:", dir, " (cached gitignore? " + bool2Str(ok) + ")")

		// Reached root path without any match
		if dir == "." {
			return false
		}

		dir = filepath.Dir(dir)
	}
}

// Walks the directory at path, returning a list of all the relative file paths.
// Ignores files and folders named ".git".
//
// Returns:
// paths is a list of paths relative to path.
// gitIgnorePaths is a list of absolute paths of all the .gitignore files found.
func getPathsRecursivelyRelativeTo(ctx context.Context, path string) (paths, gitIgnorePaths []string, err error) {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}

	paths = make([]string, 0)
	gitIgnorePaths = make([]string, 0)

	err = myWalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err != nil {
				return nil
			}

			if d.Name() == ".git" {
				if d.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}

			// What if .gitignore is suddenly a symlink or something?
			if !d.IsDir() && d.Name() == ".gitignore" {
				gitIgnorePaths = append(gitIgnorePaths, filePath)
			}

			// Skips when filePath == path, required to prevent out-of-bounds below.
			if len(filePath) <= len(path) {
				return nil
			}

			if d.IsDir() {
				paths = append(paths, filePath[len(path)+1:]+"/")
			} else {
				paths = append(paths, filePath[len(path)+1:])
			}
			return nil
		}
	})

	if err != nil {
		return nil, nil, err
	}

	return paths, gitIgnorePaths, nil
}

// Returns the index to the next folder.
// If none were found, it returns an error.
// Assumptions:
// - The paths are in the order of filepath.WalkDir.
// - That folders end in a single forward slash '/' character.
// - index is a valid index into paths.
func skipDir(paths []string, index int) (int, error) {
	// On errors, we return -2 just in case you chose not to handle the error.
	// Because when your for loop then increments the index, it will still be negative (-1) and crash.
	errorIndex := -2

	dirToSkip := filepath.ToSlash(filepath.Dir(paths[index]))

	if dirToSkip == "/" || dirToSkip == "." {
		return errorIndex, errors.New("skipping the whole root")
	}

	// filepath.Dir() trims forward-slashes but we need it for the prefix-check.
	dirToSkip += "/"

	for i := index + 1; i < len(paths); i++ {
		if !strings.HasPrefix(paths[i], dirToSkip) {
			return i, nil
		}
	}

	return errorIndex, errors.New("reached the end of paths")
}

func untrackedPathsNotIgnoredWorker(ctx context.Context, paths []string, ignoresCache map[string]*ignore.GitIgnore, indexEntries map[string]GitIndexEntry, gitLinkPaths []string, respectGitIgnore bool) map[string]ChangedFile {
	out := make(map[string]ChangedFile)

	// We can not use `for i := range paths` here, because then we wouldn't be able to reassign the index variable i.
loop:
	for i := 0; i < len(paths); i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Path relative to the repository folder e.g. "src/file.cpp"
			rel := paths[i]

			// If it's in the .git/index, it's tracked
			_, tracked := indexEntries[filepath.ToSlash(rel)]
			if tracked {
				continue
			}

			// Skip submodules (gitlinks)
			for _, path := range gitLinkPaths {
				if strings.HasPrefix(rel, path) {
					if gogitstatus_debug_ignored {
						fmt.Println("IGNORED (because submodule/gitlink):", rel)
					}
					continue loop
				}
			}

			isDir := rel[len(rel)-1] == '/' // We added this '/' manually in getPathsRecursivelyRelativeTo(), so no cross-platform worries.

			// Don't add ignored files
			if respectGitIgnore && ignoreMatch(rel, ignoresCache) {
				if gogitstatus_debug_ignored {
					fmt.Println("IGNORED:", rel)
				}

				// Skip ignored directories.
				// This is not strictly necessary since we always check
				// if any parent folders are ignored, but it avoids unnecessary work
				if isDir && !gogitstatus_debug_disable_skipdir {
					var err error
					if gogitstatus_debug_skipdir {
						fmt.Println("SKIPPING FROM:", paths[i])
					}
					i, err = skipDir(paths, i) // PERF: Could pass rel to avoid reading it again in the body of the function

					i -= 1 // Since the next iteration will increment i, make sure it's going to be the correct value.

					if err != nil {
						if gogitstatus_debug_skipdir {
							fmt.Println("SKIPPED TO: END")
						}
						break loop
					}
					if gogitstatus_debug_skipdir {
						fmt.Println("SKIPPED TO: ", paths[i+1])
					}
				}
			} else if !isDir { // Don't add directories
				out[rel] = ChangedFile{Untracked: true}
			}
		}
	}

	return out
}

// Returns untracked files that aren't ignored.
// It recursively iterates through the directory path, ignoring files/directories named ".git" and files ignored by .gitignore
func untrackedPathsNotIgnored(ctx context.Context, paths []string, gitIgnorePaths []string, path string, indexEntries map[string]GitIndexEntry, respectGitIgnore bool, numCPUs int) (map[string]ChangedFile, error) {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}

	start := time.Now()
	// Compile all the .gitignore files we found during the directory walk
	ignoresCache := make(map[string]*ignore.GitIgnore)
	if respectGitIgnore {
		for _, gitIgnorePath := range gitIgnorePaths {
			ignore, err := ignore.CompileIgnoreFile(gitIgnorePath)
			if err == nil {
				// The root folder key ends up being "." in the ignoresCache
				// because filepath.Dir("") == "."
				pathLookup := filepath.Dir(gitIgnorePath[len(path)+1:])
				ignoresCache[pathLookup] = ignore
			}
		}
	}
	if gogitstatus_debug_profiling {
		fmt.Println("Compiling gitignore:", time.Since(start))
	}

	start = time.Now()
	// Submodule / gitlink paths. They all end in a path separator to signify being a folder
	// PERF: This could be moved to happen right after ParseGitIndex() and then pass gitLinksPath to this function
	// That way we save the ~8 milliseconds this takes (on chromium repo)
	// by putting it where we're already waiting on a serial dependency to finish (walking the filesystem).
	gitLinkPaths := make([]string, 0)
	for path, entry := range indexEntries {
		if (entry.Mode & OBJECT_TYPE_MASK) == GITLINK {
			gitLinkPaths = append(gitLinkPaths, path+string(os.PathSeparator))
		}
	}
	if gogitstatus_debug_profiling {
		fmt.Println("Creating list of gitlinks:", time.Since(start))
	}

	start = time.Now()
	slices := spreadArrayIntoSlicesForGoroutines(len(paths), numCPUs)
	results := make([]map[string]ChangedFile, numCPUs)

	var wg sync.WaitGroup
	wg.Add(len(slices))
	for threadIdx, slice := range slices {
		if gogitstatus_debug_goroutine_slices {
			fmt.Println("GOROUTINE NUMBER", threadIdx, "slice:", paths[slice.start:slice.start+slice.length])
		}
		go func(threadIdx int, slice sliceType) {
			ourSlice := paths[slice.start : slice.start+slice.length]
			results[threadIdx] = untrackedPathsNotIgnoredWorker(ctx, ourSlice, ignoresCache, indexEntries, gitLinkPaths, respectGitIgnore)
			wg.Done()
		}(threadIdx, slice)
	}
	wg.Wait()

	if gogitstatus_debug_profiling {
		fmt.Println("Ignore worker:", time.Since(start))
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		start = time.Now()
		// Merge the results
		for i := 1; i < len(results); i++ {
			for k, v := range results[i] {
				results[0][k] = v
			}
		}
		if gogitstatus_debug_profiling {
			fmt.Println("Merge results:", time.Since(start))
		}

		return results[0], nil
	}
}

// Use this function to also include directories containing unstaged/untracked files
// by passing the output of Status() or StatusWithContext() through this function.
// Does not modify the changedFiles input argument.
func IncludingDirectories(changedFiles map[string]ChangedFile) map[string]ChangedFile {
	ret := make(map[string]ChangedFile)
	for k, v := range changedFiles {
		ret[k] = v
	}

	// Bad time complexity, could maybe refactor the normal status functions
	// to include directories (indicated by a trailing path separator?) to speed it up.
	for path, e := range changedFiles {
		parent := path
		for strings.ContainsRune(parent, os.PathSeparator) {
			parent = filepath.Dir(parent)
			ret[parent] = e
		}
	}

	return ret
}

// Meant to essentially undo IncludingDirectories()
// by passing the output of IncludingDirectories() through this function.
// Use this function to exclude directories containing unstaged/untracked files.
// Does not modify the changedFiles input argument.
func ExcludingDirectories(changedFiles map[string]ChangedFile) map[string]ChangedFile {
	ret := make(map[string]ChangedFile)
	for k, v := range changedFiles {
		ret[k] = v
	}

	// Bad time complexity, could maybe refactor the normal status functions
	// to include directories (indicated by a trailing path separator?) to speed it up.
	for path := range ret {
		if !strings.ContainsRune(path, os.PathSeparator) {
			continue
		}

		parent := path
		for strings.ContainsRune(parent, os.PathSeparator) {
			parent = filepath.Dir(parent)
			delete(ret, parent)
		}
	}

	return ret
}

// Returns changed files without those that were deleted
// Does not modify the changedFiles input argument.
func ExcludingDeleted(changedFiles map[string]ChangedFile) map[string]ChangedFile {
	ret := make(map[string]ChangedFile)
	for path, e := range changedFiles {
		if e.WhatChanged&DELETED != 0 {
			continue
		}

		ret[path] = e
	}

	return ret
}

// Takes in the root path of a local git repository and returns the list of changed (unstaged/untracked) files in filepaths relative to path, or an error.
func Status(path string, numCPUsOptional ...int) (map[string]ChangedFile, error) {
	ctx := context.WithoutCancel(context.Background())
	return StatusWithContext(ctx, path, numCPUsOptional...)
}

// Cancellable with context, takes in the root path of a local git repository and returns the list of changed (unstaged/untracked) files in filepaths relative to path, or an error.
func StatusWithContext(ctx context.Context, path string, numCPUsOptional ...int) (map[string]ChangedFile, error) {
	dotGitPath := myJoin(path, ".git")
	stat, err := os.Stat(dotGitPath)
	if err != nil || !stat.IsDir() {
		return nil, errors.New("not a Git repository")
	}

	return StatusRaw(ctx, path, myJoin(dotGitPath, "index"), true, numCPUsOptional...)
}

type sliceType struct {
	start  int
	length int
}

// This function and Slice struct were copied from util.go in my fen file manager
// See: https://github.com/kivattt/fen/blob/main/util.go#L1286
func spreadArrayIntoSlicesForGoroutines(arrayLength, numGoroutines int) []sliceType {
	if arrayLength == 0 {
		return []sliceType{}
	}

	if numGoroutines <= 1 {
		return []sliceType{
			{0, arrayLength},
		}
	}

	// More goroutines than there are elements, use arrayLength goroutines instead.
	// That is, 1 goroutine per element...
	if numGoroutines >= arrayLength {
		var result []sliceType
		for i := 0; i < arrayLength; i++ {
			result = append(result, sliceType{i, 1})
		}
		return result
	}

	var result []sliceType
	lengthPerGoroutine := arrayLength / numGoroutines

	rollingIndex := 0
	for i := 0; i < numGoroutines-1; i++ {
		result = append(result, sliceType{
			start:  rollingIndex,
			length: lengthPerGoroutine,
		})

		rollingIndex += lengthPerGoroutine
	}

	// Last goroutine will handle the last part of the array
	result = append(result, sliceType{
		start:  rollingIndex,
		length: arrayLength - rollingIndex,
	})

	return result
}

func trackedPathsChanged(ctx context.Context, path string, indexEntries map[string]GitIndexEntry, numCPUs int) (map[string]ChangedFile, error) {
	outs := make([]map[string]ChangedFile, numCPUs)
	for i := range outs {
		outs[i] = make(map[string]ChangedFile)
	}

	// Turn indexEntries into a list so we can iterate over it spread across threads easily
	type IndexEntry struct {
		path  string
		entry GitIndexEntry
	}

	indexEntriesSlice := make([]IndexEntry, 0, len(indexEntries))
	for k, v := range indexEntries {
		indexEntriesSlice = append(indexEntriesSlice, IndexEntry{
			path:  k,
			entry: v,
		})
	}

	splits := spreadArrayIntoSlicesForGoroutines(len(indexEntriesSlice), numCPUs)

	var wg sync.WaitGroup
	wg.Add(len(splits))

	for threadIdx, split := range splits {
		go func(threadIdx int, split sliceType) {
			defer wg.Done()

			for i := split.start; i < split.start+split.length; i++ {
				select {
				case <-ctx.Done():
					return
				default:
					e := indexEntriesSlice[i]
					entryPath := e.path
					entry := e.entry

					entryPathFromSlash := filepath.FromSlash(entryPath)
					// Faster than filepath.Join()
					fullPath := path + string(os.PathSeparator) + entryPathFromSlash

					stat, statErr := os.Lstat(fullPath)
					if statErr != nil {
						outs[threadIdx][entryPathFromSlash] = ChangedFile{WhatChanged: DELETED, Untracked: false}
					} else {
						whatChanged := fileChanged(entry, fullPath, stat)
						if whatChanged != 0 {
							outs[threadIdx][entryPathFromSlash] = ChangedFile{WhatChanged: whatChanged, Untracked: false}
						}
					}
				}
			}
		}(threadIdx, split)
	}

	wg.Wait()

	// Merge the results into the first element
	for i := 1; i < len(outs); i += 1 {
		for k, v := range outs[i] {
			outs[0][k] = v
		}
	}

	return outs[0], nil
}

// Cancellable with context, does not check if path is a valid git repository.
func StatusRaw(ctx context.Context, path string, gitIndexPath string, respectGitIgnore bool, numCPUsOptional ...int) (map[string]ChangedFile, error) {
	var numCPUs int
	if len(numCPUsOptional) > 0 {
		numCPUs = numCPUsOptional[0]
	} else {
		numCPUs = runtime.NumCPU()
	}

	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		return nil, errors.New("path does not exist: " + path)
	}

	var paths []string
	var gitIgnorePaths []string
	var walkDirError error
	var walkDirWaitGroup sync.WaitGroup
	walkDirWaitGroup.Add(1)
	go func() {
		start := time.Now()
		// Walk the directory recursively in a single thread
		paths, gitIgnorePaths, walkDirError = getPathsRecursivelyRelativeTo(ctx, path)
		if gogitstatus_debug_profiling {
			fmt.Println("Walking:", time.Since(start))
		}
		walkDirWaitGroup.Done()
	}()

	// If .git/index file is missing, all files are unstaged/untracked
	_, err = os.Stat(gitIndexPath)
	if err != nil {
		walkDirWaitGroup.Wait()
		if walkDirError != nil {
			return nil, walkDirError
		}
		return untrackedPathsNotIgnored(ctx, paths, gitIgnorePaths, path, make(map[string]GitIndexEntry), respectGitIgnore, numCPUs)
	}

	start := time.Now()
	indexEntries, err := ParseGitIndex(ctx, gitIndexPath)
	if gogitstatus_debug_profiling {
		fmt.Println("ParseGitIndex:", time.Since(start))
	}
	if err != nil {
		return nil, errors.New("unable to read " + gitIndexPath + ": " + err.Error())
	}

	var untrackedPaths map[string]ChangedFile
	var pathsErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		walkDirWaitGroup.Wait()
		if walkDirError != nil {
			pathsErr = walkDirError
			return
		}

		untrackedPaths, pathsErr = untrackedPathsNotIgnored(ctx, paths, gitIgnorePaths, path, indexEntries, respectGitIgnore, numCPUs)
	}()

	start = time.Now()
	out, err := trackedPathsChanged(ctx, path, indexEntries, numCPUs)
	if gogitstatus_debug_profiling {
		fmt.Println("Tracked:", time.Since(start))
	}
	if err != nil {
		return nil, err
	}

	wg.Wait()

	if pathsErr != nil {
		return nil, pathsErr
	}

	// Add untracked files
	for k, v := range untrackedPaths {
		out[k] = v
	}

	return out, nil
}
