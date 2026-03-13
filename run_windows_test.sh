if ! command -v wine >/dev/null; then
	echo "wine not installed. Maybe try:"
	echo "    sudo apt install wine"
	exit 1
fi

bin=./test-binaries

if [ ! -d $bin ]; then
	mkdir $bin
fi

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o $bin/windows-amd64

wine $bin/windows-amd64
