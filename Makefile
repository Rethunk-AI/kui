.PHONY: all build test vet web web-a11y coverage coverage-report web-coverage
all: web build test vet

web:
	cd web && corepack yarn install && corepack yarn run build

web-a11y: web
	cd web && corepack yarn run a11y:ci

BUILD_TAGS ?= libvirt

build:
	go build -tags $(BUILD_TAGS) -o bin/ ./cmd/...

test:
	go test -tags $(BUILD_TAGS) ./...

vet:
	go vet -tags $(BUILD_TAGS) ./...

coverage:
	go test -tags $(BUILD_TAGS) -coverprofile=coverage.out ./...

coverage-report: coverage
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

web-coverage:
	cd web && corepack yarn install && corepack yarn run test:coverage
