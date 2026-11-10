.PHONY: all build build-windows build-linux test test-mesh demo clean

BINARY_NAME=wireshell

all: test build

build:
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/wireshell

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME).exe ./cmd/wireshell

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 ./cmd/wireshell

test:
	go test -v ./...

# 🚀 One command to build, launch, and test P2P connection & hole punching
demo: build-windows
	./wireshell.exe test-mesh

test-mesh:
	go run ./cmd/wireshell test-mesh

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-linux-amd64
