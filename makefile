VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X skate/internal/version.Version=$(VERSION)

.PHONY: build install lint test clean

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o skate ./cmd/skate

install: build test
	mkdir -p ~/.local/bin/
	install -m 0755 skate ~/.local/bin/

lint:
	golangci-lint run ./...

test:
	go test ./...

test-race:
	CGO_ENABLED=1 go test -race ./...

clean:
	rm -f skate
