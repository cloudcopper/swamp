.PHONY:all
all: swamp

SRC = $(shell find . -path '*.go')
swamp: ${SRC}
	CGO_ENABLED=0 go build -ldflags="-s -w" ./cmd/swamp

.PHONY:test
test:
	CGO_ENABLED=0 go test ./...
