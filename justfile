export CGO_ENABLED := "0"
export GOARCH := "amd64"
export GOOS := "linux"

build:
  mkdir -p build
  go build -o build/coredock

