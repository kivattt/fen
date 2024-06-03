#!/usr/bin/env bash
bin=./bin

if [ ! -d $bin ]; then
	mkdir $bin
fi

GOOS=linux GOARCH=amd64 go build -o $bin/fen-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o $bin/fen-macos-amd64
GOOS=freebsd GOARCH=amd64 go build -o $bin/fen-freebsd-amd64
GOOS=windows GOARCH=amd64 go build -o $bin/fen-windows-amd64.exe
