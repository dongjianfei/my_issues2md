.PHONY: build test test-integration lint clean

build:
	go build -o bin/issue2md ./cmd/issue2md/

test:
	go test ./... -v -count=1

test-integration:
	go test ./... -v -count=1 -tags=integration

lint:
	go vet ./...

clean:
	rm -rf bin/
