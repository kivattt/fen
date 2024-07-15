#!/usr/bin/env bash
bin=./bin

if [ ! -d $bin ]; then
	mkdir $bin
fi

ldflags="-s -w"

# amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fen-linux-amd64
GOOS=darwin GOARCH=amd64 go build -ldflags="$ldflags" -o $bin/fen-macos-amd64
GOOS=freebsd GOARCH=amd64 go build -ldflags="$ldflags" -o $bin/fen-freebsd-amd64
GOOS=windows GOARCH=amd64 go build -ldflags="$ldflags" -o $bin/fen-windows-amd64.exe

# i386
GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fen-linux-i386
GOOS=freebsd GOARCH=386 go build -ldflags="$ldflags" -o $bin/fen-freebsd-i386
GOOS=windows GOARCH=386 go build -ldflags="$ldflags" -o $bin/fen-windows-i386.exe

# arm64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fen-linux-arm64
GOOS=darwin GOARCH=arm64 go build -ldflags="$ldflags" -o $bin/fen-macos-arm64
GOOS=freebsd GOARCH=arm64 go build -ldflags="$ldflags" -o $bin/fen-freebsd-arm64
GOOS=windows GOARCH=arm64 go build -ldflags="$ldflags" -o $bin/fen-windows-arm64.exe

# arm
GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -ldflags="$ldflags" -o $bin/fen-linux-arm
GOOS=freebsd GOARCH=arm go build -ldflags="$ldflags" -o $bin/fen-freebsd-arm
GOOS=windows GOARCH=arm go build -ldflags="$ldflags" -o $bin/fen-windows-arm.exe
