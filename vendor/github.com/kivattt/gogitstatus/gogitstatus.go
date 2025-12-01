package gogitstatus

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sabhiram/go-gitignore"
)

// A small subset of a Git index entry
type GitIndexEntry struct {
	MetadataChangedTimeSeconds     uint32 // ctime
	MetadataChangedTimeNanoSeconds uint32 // ctime
	ModifiedTimeSeconds            uint32
	ModifiedTimeNanoSeconds        uint32
	Mode                           uint32 // Contains the file type and unix permission bits
	Hash                           []byte // 20 bytes for the standard SHA-1
}

// This function is only used for path lengths in the .git/index longer than 0xffe bytes
// TODO: Can speed this up by first reading 0xfff bytes, and then 8 bytes at a time until the last byte of an 8-byte section is a null byte
func readIndexEntryPathName(reader *bytes.Reader) (strings.Builder, error) {
	var ret strings.Builder

	// Entry length so far
	entryLength := 40 + 20 + 2

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

	reader := bytes.NewReader(data)

	headerBytes := make([]byte, 12)
	_, err = io.ReadFull(reader, headerBytes)
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
	entries := make(map[string]GitIndexEntry)

	var entryIndex uint32
	for entryIndex = 0; entryIndex < numEntries; entryIndex++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Read 64-bit metadata changed time
			cTimeBytes := make([]byte, 8) // 64 bits
			if _, err := io.ReadFull(reader, cTimeBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 64-bit metadata changed time (ctime) within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			ctimeSeconds := binary.BigEndian.Uint32(cTimeBytes[:4])
			ctimeNanoSeconds := binary.BigEndian.Uint32(cTimeBytes[4:])

			// Read 64-bit modified time
			mTimeBytes := make([]byte, 8) // 64 bits
			if _, err := io.ReadFull(reader, mTimeBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 64-bit modified time within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			mTimeSeconds := binary.BigEndian.Uint32(mTimeBytes[:4])
			mTimeNanoSeconds := binary.BigEndian.Uint32(mTimeBytes[4:])

			// Seek to 32-bit mode
			if _, err := reader.Seek(8, 1); err != nil { // 64 bits
				return nil, errors.New("invalid size, unable to seek to 32-bit mode within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			// Read 32-bit mode
			bytes := make([]byte, 4) // 32 bits
			if _, err := io.ReadFull(reader, bytes); err != nil {
				return nil, errors.New("invalid size, unable to read 32-bit mode within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			mode := binary.BigEndian.Uint32(bytes)

			// Seek to "object name" (hash data)
			if _, err := reader.Seek(12, 1); err != nil { // 96 bits
				return nil, errors.New("invalid size, unable to seek to object name within entry at index " + strconv.FormatInt(int64(entryIndex), 10))
			}

			// Read hash data
			hash := make([]byte, 20) // 160 bits
			if _, err := io.ReadFull(reader, hash); err != nil {
				return nil, errors.New("invalid size, unable to read 20-byte SHA-1 hash at index " + strconv.FormatUint(uint64(entryIndex), 10))
			}

			flagsBytes := make([]byte, 2) // 16 bits 'flags' field
			if _, err := io.ReadFull(reader, flagsBytes); err != nil {
				return nil, errors.New("invalid size, unable to read 2-byte flags field at index " + strconv.FormatUint(uint64(entryIndex), 10))
			}

			flags := binary.BigEndian.Uint16(flagsBytes)
			nameLength := flags & 0xfff

			var pathName strings.Builder
			if nameLength == 0xfff { // Path name length >= 0xfff, need to manually find null bytes
				// Read variable-length path name
				pathName, err = readIndexEntryPathName(reader)
				if err != nil {
					return nil, err
				}
			} else {
				bytes := make([]byte, nameLength)
				if _, err := io.ReadFull(reader, bytes); err != nil {
					return nil, errors.New("invalid size, unable to read path name of size " + strconv.FormatUint(uint64(nameLength), 10) + " at index " + strconv.FormatUint(uint64(entryIndex), 10))
				}

				pathName.Write(bytes)
				entryLength := 40 + 20 + 2 // Entry length so far
				// Read up to 8 null padding bytes
				n := 8 - ((int(nameLength) + entryLength) % 8)
				if n == 0 {
					n = 8
				}

				b := make([]byte, n)
				if _, err = io.ReadFull(reader, b); err != nil {
					return nil, errors.New("invalid size, unable to read path name null bytes of size " + strconv.FormatUint(uint64(n), 10) + " at index " + strconv.FormatUint(uint64(entryIndex), 10))
				}

				for _, e := range b {
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
				Hash:                           hash,
			}
		}
	}

	return entries, nil
}

func hashMatches(path string, stat os.FileInfo, hash []byte) bool {
	// Symlinks are hashed with the target path, not the data of the target file
	// On Windows, symlinks are stored as regular files (with target path as the file data), so we handle them as such later
	if runtime.GOOS != "windows" && stat.Mode()&os.ModeSymlink != 0 /*|| !stat.Mode().IsRegular()*/ {
		newHash := sha1.New()
		targetPath, err := os.Readlink(path)
		if err != nil {
			return false
		}

		newHash.Write(append([]byte("blob "+strconv.Itoa(len(targetPath))), 0))
		newHash.Write([]byte(targetPath))
		return reflect.DeepEqual(hash, newHash.Sum(nil))
	}

	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	newHash := sha1.New()
	_, err = newHash.Write(append([]byte("blob "+strconv.FormatInt(stat.Size(), 10)), 0))
	if err != nil {
		return false
	}

	data, err := openFileData(file, stat)
	if err != nil {
		return false
	}
	defer closeFileData(data)

	newHash.Write(data)

	// Debugging
	/*bool2Str := func(b bool) string {
		if b {
			return "\x1b[32m true\x1b[0m"
		}
		return "\x1b[31mfalse\x1b[0m"
	}

	b := reflect.DeepEqual(hash, newHash.Sum(nil))
	fmt.Println("hash: " + hex.EncodeToString(hash) + ", newHash: " + hex.EncodeToString(newHash.Sum(nil)) + ", matches? " + bool2Str(b) + ", " + path)*/

	return reflect.DeepEqual(hash, newHash.Sum(nil))
}

type WhatChanged int

const (
	// https://github.com/git/git/blob/ef8ce8f3d4344fd3af049c17eeba5cd20d98b69f/statinfo.h#L35
	MTIME_CHANGED WhatChanged = 0x0001 // We don't use this
	CTIME_CHANGED WhatChanged = 0x0002
	OWNER_CHANGED WhatChanged = 0x0004
	MODE_CHANGED  WhatChanged = 0x0008
	INODE_CHANGED WhatChanged = 0x0010 // Use or not?
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

// If you pass this a nil value for stat, it will return 0
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

	if !hashMatches(entryFullPath, stat, entry.Hash) {
		whatChanged |= DATA_CHANGED
	}

	return whatChanged
}

type ChangedFile struct {
	WhatChanged WhatChanged
	Untracked   bool // true = Untracked, false = Unstaged
}

func ignoreMatch(path string, ignoresMap map[string]*ignore.GitIgnore) bool {
	dir := filepath.Dir(path)
	for {
		ignore, ok := ignoresMap[dir]
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return false
		}

		if ok && ignore.MatchesPath(rel) {
			return true
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

// Recursively iterates through the directory path, returning a list of all the filepaths found, ignoring files/directories named ".git" and untracked files ignored by .gitignore
func AccumulatePathsNotIgnored(ctx context.Context, path string, indexEntries map[string]GitIndexEntry, respectGitIgnore bool) (map[string]ChangedFile, error) {
	ignoresMap := make(map[string]*ignore.GitIgnore)

	paths := make(map[string]ChangedFile)
	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err != nil {
				return nil
			}

			// We need to handle everything based on the path relative to path
			rel, err := filepath.Rel(path, filePath)
			if err != nil {
				return nil
			}

			// If it's in the .git/index, it's tracked
			_, tracked := indexEntries[filepath.ToSlash(rel)]

			// Don't add untracked ignored files
			if respectGitIgnore && !tracked {
				if d.IsDir() {
					childIgnore, err := ignore.CompileIgnoreFile(filepath.Join(filePath, ".gitignore"))
					if err == nil {
						ignoresMap[rel] = childIgnore
					}
				}

				if rel == "." {
					return nil
				}

				if ignoreMatch(rel, ignoresMap) {
					if d.IsDir() {
						return filepath.SkipDir
					} else {
						return nil
					}
				}
			}

			// git status seems to ignore any file/directory named ".git", regardless of its parent directory
			if d.Name() == ".git" {
				if d.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}

			if d.IsDir() {
				return nil
			}

			paths[rel] = ChangedFile{Untracked: !tracked}
			return nil
		}
	})

	if err != nil {
		return nil, err
	}

	return paths, nil
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
func Status(path string) (map[string]ChangedFile, error) {
	ctx := context.WithoutCancel(context.Background())
	return StatusWithContext(ctx, path)
}

// Cancellable with context, takes in the root path of a local git repository and returns the list of changed (unstaged/untracked) files in filepaths relative to path, or an error.
func StatusWithContext(ctx context.Context, path string) (map[string]ChangedFile, error) {
	dotGitPath := filepath.Join(path, ".git")
	stat, err := os.Stat(dotGitPath)
	if err != nil || !stat.IsDir() {
		return nil, errors.New("not a Git repository")
	}

	return StatusRaw(ctx, path, filepath.Join(dotGitPath, "index"), true)
}

// Cancellable with context, does not check if path is a valid git repository
func StatusRaw(ctx context.Context, path string, gitIndexPath string, respectGitIgnore bool) (map[string]ChangedFile, error) {
	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		return nil, errors.New("path does not exist: " + path)
	}

	// If .git/index file is missing, all files are unstaged/untracked
	_, err = os.Stat(gitIndexPath)
	if err != nil {
		return AccumulatePathsNotIgnored(ctx, path, make(map[string]GitIndexEntry), respectGitIgnore)
	}

	indexEntries, err := ParseGitIndex(ctx, gitIndexPath)
	if err != nil {
		return nil, errors.New("unable to read " + gitIndexPath + ": " + err.Error())
	}

	paths, err := AccumulatePathsNotIgnored(ctx, path, indexEntries, respectGitIgnore)
	if err != nil {
		return nil, err
	}

	// Filter unchanged files
	for p, entry := range indexEntries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			pFromSlash := filepath.FromSlash(p)
			fullPath := filepath.Join(path, pFromSlash)

			_, pathFound := paths[pFromSlash]

			stat, statErr := os.Lstat(fullPath)
			if statErr != nil {
				stat = nil // Just to be sure

				if pathFound { // Deleted file
					delete(paths, pFromSlash)
					continue
				} else {
					// File is tracked but ignored, so we didn't add it previously. This might cause bugs?

					// Deleted files need to be added since we previously only added files that already exist on the filesystem
					paths[pFromSlash] = ChangedFile{WhatChanged: DELETED, Untracked: false}
					continue
				}
			}

			whatChanged := fileChanged(entry, fullPath, stat)
			if whatChanged == 0 {
				delete(paths, pFromSlash)
			} else {
				paths[pFromSlash] = ChangedFile{WhatChanged: whatChanged, Untracked: false}
			}
		}
	}

	return paths, nil
}
