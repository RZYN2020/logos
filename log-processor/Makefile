# Makefile for Log Processor CI/CD

.PHONY: all build test lint clean docker-build docker-push bench vet fmt check

# Variables
BINARY_NAME=log-processor
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"
DOCKER_IMAGE ?= ghcr.io/log-system/log-processor/log-processor
GO_VERSION ?= 1.21

# Default target
all: lint test build

# ==================== Build ====================
build:
	@echo "Building ${BINARY_NAME}..."
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY_NAME} cmd/job/main.go

build-linux:
	@echo "Building ${BINARY_NAME} for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 cmd/job/main.go

build-darwin:
	@echo "Building ${BINARY_NAME} for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 cmd/job/main.go

build-windows:
	@echo "Building ${BINARY_NAME} for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe cmd/job/main.go

# ==================== Test ====================
test:
	@echo "Running tests..."
	go test -v -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-verbose:
	@echo "Running verbose tests..."
	go test -v -count=1 ./...

bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -benchtime=1s ./...

bench-profile:
	@echo "Running benchmarks with profiling..."
	go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...

# ==================== Lint ====================
lint:
	@echo "Running linter..."
	golangci-lint run ./...

lint-fix:
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix ./...

vet:
	@echo "Running go vet..."
	go vet ./...

fmt:
	@echo "Running go fmt..."
	go fmt ./...

check: fmt vet lint
	@echo "All checks passed!"

# ==================== Dependencies ====================
deps:
	@echo "Downloading dependencies..."
	go mod download

deps-tidy:
	@echo "Tidying dependencies..."
	go mod tidy

deps-verify:
	@echo "Verifying dependencies..."
	go mod verify

# ==================== Clean ====================
clean:
	@echo "Cleaning..."
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME}-*
	rm -f coverage.out coverage.html
	rm -f *.prof
	go clean -cache

# ==================== Docker ====================
docker-build:
	@echo "Building Docker image..."
	docker build -t ${DOCKER_IMAGE}:${VERSION} -f Dockerfile .

docker-push:
	@echo "Pushing Docker image..."
	docker push ${DOCKER_IMAGE}:${VERSION}

docker-run:
	@echo "Running Docker container..."
	docker run --rm ${DOCKER_IMAGE}:${VERSION}

# ==================== Kubernetes ====================
k8s-apply:
	@echo "Applying Kubernetes manifests..."
	kubectl apply -f deploy/kubernetes/

k8s-delete:
	@echo "Deleting Kubernetes resources..."
	kubectl delete -f deploy/kubernetes/

k8s-status:
	@echo "Kubernetes deployment status:"
	kubectl get deployments
	kubectl get pods

# ==================== Release ====================
release: check build-linux build-darwin build-windows
	@echo "Creating release artifacts..."
	mkdir -p dist
	mv ${BINARY_NAME}-linux-amd64 dist/
	mv ${BINARY_NAME}-darwin-amd64 dist/
	mv ${BINARY_NAME}-windows-amd64.exe dist/
	@echo "Release artifacts created in dist/"

release-tag:
	@echo "Creating release tag..."
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)

# ==================== Help ====================
help:
	@echo "Log Processor Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all            - Run lint, test, and build (default)"
	@echo "  build          - Build binary for current platform"
	@echo "  build-linux    - Build binary for Linux"
	@echo "  build-darwin   - Build binary for macOS"
	@echo "  build-windows  - Build binary for Windows"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-verbose   - Run verbose tests"
	@echo "  bench          - Run benchmarks"
	@echo "  bench-profile  - Run benchmarks with profiling"
	@echo "  lint           - Run linter"
	@echo "  lint-fix       - Run linter with auto-fix"
	@echo "  vet            - Run go vet"
	@echo "  fmt            - Run go fmt"
	@echo "  check          - Run all checks (fmt, vet, lint)"
	@echo "  deps           - Download dependencies"
	@echo "  deps-tidy      - Tidy dependencies"
	@echo "  deps-verify    - Verify dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Push Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  k8s-apply      - Apply Kubernetes manifests"
	@echo "  k8s-delete     - Delete Kubernetes resources"
	@echo "  k8s-status     - Show Kubernetes status"
	@echo "  release        - Create release artifacts"
	@echo "  release-tag    - Create and push release tag"
	@echo "  help           - Show this help message"
