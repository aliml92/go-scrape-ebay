# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFILES=$(wildcard *.go) $(wildcard cmd/goscrapeebay/*.go)

# Binary name and path
BINARY_NAME=goscrapeebay
BINARY_PATH=./cmd/goscrapeebay

# Linter
GOLINT=staticcheck

# Formatter
GOFUMPT=gofumpt

.PHONY: all build clean test run deps install lint fmt

all: deps lint fmt build

build:
	cd $(BINARY_PATH) && $(GOBUILD) -o ../../$(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) ./... && cd $(BINARY_PATH) && $(GOTEST) ./...

run: build
	./$(BINARY_NAME)

deps:
	$(GOMOD) download

install:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install mvdan.cc/gofumpt@latest

lint:
	$(GOLINT) ./...
	$(GOLINT) $(BINARY_PATH)/...

fmt:
	$(GOFUMPT) -l -w .
	$(GOFUMPT) -l -w $(BINARY_PATH)

# Cross compilation
build-linux:
	cd $(BINARY_PATH) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o ../../$(BINARY_NAME) -v

build-windows:
	cd $(BINARY_PATH) && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o ../../$(BINARY_NAME).exe -v

build-mac:
	cd $(BINARY_PATH) && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o ../../$(BINARY_NAME)_mac -v