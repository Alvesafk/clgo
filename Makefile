SHELL := /bin/sh

BINARY := clgo
VERSION ?= dev
DIST := dist
LDFLAGS := -s -w -X github.com/Alvesafk/clgo/cmd.Version=$(VERSION)

.PHONY: help fmt lint test test-cover test-race test-benchmark vet staticcheck bench coverage build release-local clean

help:
	@printf '%s\n' \
		'make fmt            format Go sources' \
		'make lint           run linter' \
		'make test           run unit tests' \
		'make test-cover     run tests and show coverage' \
		'make test-race      run tests with race detector' \
		'make test-benchmark run benchmark' \
		'make vet            run go vet' \
		'make staticcheck    run staticcheck when installed' \
		'make bench          run benchmarks' \
		'make coverage       write dist/coverage.html' \
		'make build          build the host binary' \
		'make release-local  build Linux, Windows, and macOS archives locally' \
		'make clean          remove generated artifacts'

fmt:
	gofmt -s -w $$(find . -name '*.go' -type f)

lint:
	golangci-lint run ./...

test:
	go test ./...

test-cover:
	go test --cover ./...

test-race:
	go test -race ./...

test-benchmark:
	go test ./internal/cloc -run . -bench . -benchmem

vet:
	go vet ./...

staticcheck:
	@command -v staticcheck >/dev/null 2>&1 || { echo 'staticcheck is not installed' >&2; exit 1; }
	staticcheck ./...

bench:
	go test -bench=. -benchmem ./...

coverage:
	mkdir -p $(DIST)
	go test -coverprofile=$(DIST)/coverage.out ./...
	go tool cover -html=$(DIST)/coverage.out -o $(DIST)/coverage.html

build:
	mkdir -p bin
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) .

release-local: clean
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY)_linux_amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY)_linux_arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY)_windows_amd64.exe .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY)_darwin_amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY)_darwin_arm64 .
	cd $(DIST) && if command -v sha256sum >/dev/null 2>&1; then sha256sum $(BINARY)_* > checksums.txt; else shasum -a 256 $(BINARY)_* > checksums.txt; fi

clean:
	rm -rf bin $(DIST)
