.PHONY:all
all: swamp

SRC = $(shell find . -path '*.go' -or -path '*.html')
swamp: ${SRC}
	CGO_ENABLED=0 go build -ldflags="-s -w" ./cmd/swamp
	CGO_ENABLED=0 go build -ldflags="-s -w" ./cmd/swamp-ui-dev

.PHONY:test
test:
	CGO_ENABLED=0 go test ./...

.PHONY:coverage
coverage:
	CGO_ENABLED=0 go test -coverprofile cover.out ./...
	go tool cover -html cover.out -o cover.html

.PHONY: crit
crit:
	gocritic check -enableAll ./...
