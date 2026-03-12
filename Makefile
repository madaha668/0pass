.PHONY: build test lint cover clean

build:
	go build -o 0pass .

test:
	go test ./... -count=1 -timeout 60s

cover:
	go test ./... -count=1 -timeout 60s -coverprofile=cover.out
	go tool cover -func=cover.out

lint:
	golangci-lint run ./...

clean:
	rm -f 0pass cover.out
