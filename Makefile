.PHONY: all build test vet web
all: web build test vet

web:
	cd web && corepack yarn install && corepack yarn run build

build:
	go build -o bin/ ./cmd/...

test:
	go test ./...

vet:
	go vet ./...
