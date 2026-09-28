.PHONY: test vet build docker run-api run-worker tidy fmt

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

docker:
	docker build --target api -t securelink-api .
	docker build --target worker -t securelink-worker .

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

tidy:
	go mod tidy

fmt:
	go fmt ./...
