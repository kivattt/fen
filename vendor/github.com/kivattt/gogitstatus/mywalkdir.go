package gogitstatus

/*
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// All the code was ripped from the Go standard library code, version go1.25.6

type dirEntry = fs.DirEntry

// ReadDir reads the named directory,
// returning all its directory entries.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func myReadDir(name string) ([]dirEntry, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.ReadDir(-1)
}

// walkDir recursively descends path, calling walkDirFn.
func walkDir(path string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if err := walkDirFn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}
		return err
	}

	//dirs, err := os.ReadDir(path)
	dirs, err := myReadDir(path) // My non-sorting version
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(path, d, err)
		if err != nil {
			if err == filepath.SkipDir && d.IsDir() {
				err = nil
			}
			return err
		}
	}

	for _, d1 := range dirs {
		//path1 := filepath.Join(path, d1.Name())
		path1 := myJoin(path, d1.Name())
		if err := walkDir(path1, d1, walkDirFn); err != nil {
			if err == filepath.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the [fs.WalkDirFunc] documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but requires WalkDir to read an entire directory into memory before proceeding
// to walk that directory.
//
// WalkDir does not follow symbolic links.
//
// WalkDir calls fn with paths that use the separator character appropriate
// for the operating system. This is unlike [io/fs.WalkDir], which always
// uses slash separated paths.
func myWalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(root, fs.FileInfoToDirEntry(info), fn)
	}
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	return err
}

func myJoin(elem ...string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(elem...)
	} else {
		// If there's a bug here, fix the logic in ./path_plan9.go too.
		for i, e := range elem {
			if e != "" {
				//return filepath.Clean(strings.Join(elem[i:], string(os.PathSeparator)))
				return strings.Join(elem[i:], string(os.PathSeparator))
			}
		}
		return ""
	}
}
