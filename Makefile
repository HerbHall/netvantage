.PHONY: build build-server build-scout test lint run-server run-scout proto clean

# Binary names
SERVER_BIN=netvantage
SCOUT_BIN=scout

# Build flags
LDFLAGS=-ldflags "-s -w"

build: build-server build-scout

build-server:
	go build $(LDFLAGS) -o bin/$(SERVER_BIN) ./cmd/netvantage/

build-scout:
	go build $(LDFLAGS) -o bin/$(SCOUT_BIN) ./cmd/scout/

test:
	go test ./...

lint:
	go vet ./...

run-server: build-server
	./bin/$(SERVER_BIN)

run-scout: build-scout
	./bin/$(SCOUT_BIN)

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/v1/*.proto

clean:
	rm -rf bin/
	go clean
