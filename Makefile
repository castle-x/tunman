.PHONY: build run test fmt vet clean install install-local install-dev build-all

BINARY_NAME=tunman
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

CMD_DIR=cmd/tunman
OUT_DIR=bin

all: build

build:
	@mkdir -p $(OUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built: $(OUT_DIR)/$(BINARY_NAME) ($(VERSION))"

run: build
	./$(OUT_DIR)/$(BINARY_NAME) $(ARGS)

install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Installed: $(GOPATH)/bin/$(BINARY_NAME)"

install-local:
	@mkdir -p ~/.local/bin
	$(GOBUILD) $(LDFLAGS) -o ~/.local/bin/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Installed: ~/.local/bin/$(BINARY_NAME)"

install-dev: build
	@mkdir -p ~/.local/bin
	@cp $(OUT_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "Installed to: ~/.local/bin/$(BINARY_NAME)"

test:
	$(GOTEST) -v ./...

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

deps:
	$(GOMOD) download
	$(GOMOD) tidy

clean:
	@rm -rf $(OUT_DIR)
	@echo "Cleaned"

build-all: build-linux build-darwin build-windows
	@echo "All platforms built -> $(OUT_DIR)/"

build-linux:
	@mkdir -p $(OUT_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)

build-darwin:
	@mkdir -p $(OUT_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)

build-windows:
	@mkdir -p $(OUT_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
