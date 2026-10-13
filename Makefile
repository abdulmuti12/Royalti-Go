.PHONY: run test build check
run:
	go run ./cmd/api
test:
	go test -race ./...
build:
	go build -buildvcs=false -o bin/api ./cmd/api
check:
	go vet ./...
	go test ./...
