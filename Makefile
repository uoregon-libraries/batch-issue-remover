.PHONY: all vet clean debug

all:
	go build -o bin/remove-issues ./cmd/remove-issues

vet:
	go vet ./...

clean:
	rm -f bin/*

debug:
	rm -f bin/remove-issues
	go build -gcflags '-N -l' -o bin/remove-issues ./cmd/remove-issues
