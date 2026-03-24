.PHONY: all build test vet web web-a11y coverage coverage-report web-coverage specs-list
all: web build test vet

# Print sorted basenames of specs/done (canonical completed-spec inventory).
specs-list:
	@ls -1 specs/done | sort

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
