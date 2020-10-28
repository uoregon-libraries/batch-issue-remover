.PHONY: all vet clean

all:
	go build -o bin/remove-issues ./cmd/remove-issues

vet:
	go vet ./...

clean:
	rm -f bin/*
