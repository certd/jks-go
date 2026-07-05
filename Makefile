.PHONY: build test lint clean release-all

BINARY=jks-go
DIST_DIR=dist
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +'%Y-%m-%d_%H:%M:%S' 2>/dev/null || Get-Date -Format 'yyyy-MM-dd_HH:mm:ss' -AsUTC)

LDFLAGS=-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

build:
	go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)$(EXT) ./src/

test:
	go test -v ./src/...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(DIST_DIR)

release-all:
	$(MAKE) release GOOS=linux GOARCH=amd64
	$(MAKE) release GOOS=linux GOARCH=arm64
	$(MAKE) release GOOS=linux GOARCH=arm GOARM=7

release:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
		go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)_$(GOOS)_$(GOARCH)$(if $(GOARM),_armv$(GOARM),)$(EXT) ./src/
