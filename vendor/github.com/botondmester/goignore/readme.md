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
    ignore := goignore.CompileIgnoreLines([]string{
        "/*",
        "!/foo",
        "/foo/*",
        "!/foo/bar",
    })

    // should print `foo/baz is ignored`
    isIgnored, _ := ignore.MatchesPath("foo/baz")

    if isIgnored {
        println("foo/baz is ignored")
    } else {
        println("foo/baz is not ignored")
    }
}
```

For more examples, refer to the [goignore_test.go](goignore_test.go) file.

## Tests

Some of this package's tests were copied from the [go-gitignore](https://github.com/sabhiram/go-gitignore) package, and were modified, corrected or extended where needed.

## Fuzzing

I have fuzzed the library for about 2 hours in total, and the fuzzer did not find any crashes in that time.
Currently fuzzing does not check if the library's output is correct.

If you want to, you can do fuzzing using these commands:
```shell
go test -fuzz FuzzStringMatch
```
or
```shell
go test -fuzz FuzzWhole
```
These are implemented at the bottom of the [tests file](goignore_test.go).
