if ! command -v wine >/dev/null; then
	echo "wine not installed. Maybe try:"
	echo "    sudo apt install wine"
	exit 1
fi

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o test-windows-amd64.exe && wine test-windows-amd64.exe
