export CGO_ENABLED := "0"
export GOARCH := "amd64"
export GOOS := "linux"

version := `git describe --tags --always 2>/dev/null || echo "dev"`

run:
  go run -ldflags '-X main.Version={{version}}' main.go

build:
  docker build -t ghcr.io/ad-on-is/coredock .

push: build
  docker build -t ghcr.io/ad-on-is/coredock:latest .
  docker build -t ghcr.io/ad-on-is/coredock:{{version}} .

  docker push ghcr.io/ad-on-is/coredock:latest
  docker push ghcr.io/ad-on-is/coredock:{{version}}

watch:
  watchexec -e go -r -- just run
