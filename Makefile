.PHONY: build build-server build-scout test lint run-server run-scout proto clean license-check

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

# Allowed licenses for dependencies (BSL 1.1 compatible)
ALLOWED_LICENSES=Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0

license-check:
	@echo "Checking dependency licenses..."
	@go-licenses check ./... --allowed_licenses=$(ALLOWED_LICENSES) \
		|| (echo "ERROR: Incompatible license detected. See go-licenses output above." && exit 1)
	@echo "All dependency licenses are compatible."

license-report:
	@go-licenses report ./... --template=csv 2>/dev/null || go-licenses csv ./... 2>/dev/null

clean:
	rm -rf bin/
	go clean
