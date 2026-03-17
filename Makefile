.PHONY: all build test vet web web-a11y
all: web build test vet

web:
	cd web && corepack yarn install && corepack yarn run build

web-a11y: web
	cd web && corepack yarn run a11y:ci

build:
	go build -o bin/ ./cmd/...

test:
	go test ./...

vet:
	go vet ./...
