# goignore

[![Go Reference](https://pkg.go.dev/badge/github.com/botondmester/goignore.svg)](https://pkg.go.dev/github.com/botondmester/goignore)

A simple but fast gitignore parser for `Go`

## Install

```shell
go get github.com/botondmester/goignore
```

## Usage

This is a simple example showing how to use the library:
```go
package main

import "github.com/botondmester/goignore"

func main() {
    ignore := goignore.CompileIgnoreLines(
        "/*",
        "!/foo",
        "/foo/*",
        "!/foo/bar",
    )

    // should print `foo/baz is ignored`
    if ignore.MatchesPath("foo/baz") {
        println("foo/baz is ignored")
    } else {
        println("foo/baz is not ignored")
    }
}
```

For more examples, refer to the [goignore\_test.go](goignore_test.go) file.

## Tests

If you're not on Windows, you can still run the tests through wine with `run_windows_test.sh` e.g. on Linux.

Some of this package's tests were copied from the [go-gitignore](https://github.com/sabhiram/go-gitignore) package, and were modified, corrected or extended where needed.

## Fuzzing

Fuzz for bugs in the library, it uses [git-check-ignore](https://git-scm.com/docs/git-check-ignore) to see where we have different output.
```shell
go test -fuzz FuzzCorrectness
```

Fuzz for crashes in `makeRuleComponent()` and `matchComponent()`
```shell
go test -fuzz FuzzMatchComponent
```

Fuzz for crashes in `CompileIgnoreLines()` and `MatchesPath()`
```shell
go test -fuzz FuzzWhole
```

These are implemented at the bottom of the [tests file](goignore_test.go).
