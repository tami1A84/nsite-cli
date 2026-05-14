APP := nsite-cli

.PHONY: build install test clean

build:
	go build -o $(APP) ./cmd/nsite-cli

install:
	go install ./cmd/nsite-cli

test:
	go test ./...

clean:
	rm -f $(APP)
