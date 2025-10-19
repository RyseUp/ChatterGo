APP=chattergo
PKG=github.com/RyseUp/ChatterGo

.PHONY: run dev tidy test fmt migrate docker

run:
	go run ./cmd/server

dev:
	GIN_MODE=debug go run ./cmd/server

migrate:
	go run ./cmd/migrator

tidy:
	go mod tidy

fmt:
	go fmt ./...

test:
	go test ./...

docker:
	docker build -t $(APP):dev .