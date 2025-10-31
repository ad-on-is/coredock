export CGO_ENABLED := "0"
export GOARCH := "amd64"
export GOOS := "linux"

build:
  mkdir -p build
  go build -o build/coredock

push: build
  docker build -t ghcr.io/ad-on-is/coredock:latest .
  docker push ghcr.io/ad-on-is/coredock:latest

watch:
  watchexec -e go -r -- go run main.go
