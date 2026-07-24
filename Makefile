VERSION := $(shell git describe --exact-match --tags HEAD 2>/dev/null || echo "unpublished")
COMMIT  := $(shell git rev-parse --short HEAD)

.PHONY: build
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)" -o bin/ucloud-sandbox-cli .

.PHONY: build-windows build-windows-amd64 build-windows-arm64
build-windows: build-windows-amd64 build-windows-arm64

build-windows-amd64: WINDOWS_ARCH := amd64
build-windows-arm64: WINDOWS_ARCH := arm64
build-windows-amd64 build-windows-arm64:
	@mkdir -p bin/windows/$(WINDOWS_ARCH)
	CGO_ENABLED=0 GOOS=windows GOARCH=$(WINDOWS_ARCH) go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)" -o bin/windows/$(WINDOWS_ARCH)/ucloud-sandbox-cli.exe .

.PHONY: test
test:
	CGO_ENABLED=0 go test -v ./...

.PHONY: clean
clean:
	rm -rf bin
