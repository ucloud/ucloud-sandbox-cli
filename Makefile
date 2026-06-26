VERSION := $(shell git describe --exact-match --tags HEAD 2>/dev/null || echo "unpublished")
COMMIT  := $(shell git rev-parse --short HEAD)

.PHONY: build
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)" -o bin/ucloud-sandbox-cli .

.PHONY: test
test:
	CGO_ENABLED=0 go test -v ./...

.PHONY: clean
clean:
	rm -rf bin
