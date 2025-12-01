set -e

if [ "$(go env GOARCH)" != amd64 ]; then
    echo 'error: this is only for amd64'
    exit 1
fi
GOAMD64='v3' go test
