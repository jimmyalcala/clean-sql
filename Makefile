BINARY_NAME=clean-sql
VERSION=1.0.0
BUILD_DIR=build

.PHONY: build test clean install uninstall release

build:
	go build -ldflags "-s -w" -o $(BINARY_NAME) .

test:
	go test -v ./...

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# Cross-compile for all platforms
release: clean
	mkdir -p $(BUILD_DIR)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	@echo "Binaries in $(BUILD_DIR)/"
