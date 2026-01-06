## gogitstatus

[![Go Reference](https://pkg.go.dev/badge/github.com/kivattt/gogitstatus.svg)](https://pkg.go.dev/github.com/kivattt/gogitstatus)
[![Go Report Card](https://goreportcard.com/badge/github.com/kivattt/gogitstatus)](https://goreportcard.com/report/github.com/kivattt/gogitstatus)

gogitstatus is a library for finding unstaged/untracked files in local Git repositories\
Tested for Linux, FreeBSD and Windows\
This library is used in my terminal file manager [fen](https://github.com/kivattt/fen)

To try out `gogitstatus.Status()`, run the showstatus program:
```console
cd showstatus
go build
./showstatus . # In any git repository
```

To try out `gogitstatus.ParseIndex()`, run the showindex program:
```console
cd showindex
go build
./showindex index
```

## Running tests
Run `go test` to run all the tests.

Run `go test -fuzz=FuzzParseGitIndexFromMemory` to fuzz for crashes in the `ParseGitIndexFromMemory()` function.

## Git Index file format resources
https://git-scm.com/docs/index-format (missing some visual separation...)\
https://github.com/git/git/blob/master/read-cache.c

## TODO
- Deal with .git files that point the real .git folder elsewhere (submodules or something)
- Support exclude file priority
- Support SHA-256
- Support other Git Index versions besides 2
