.PHONY: all vet

all:
	go build -o bin/remove-issues ./cmd/remove-issues

vet:
	go vet ./...
