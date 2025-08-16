# Buffkit Makefile

.PHONY: help
help: ## Show this help message
	@echo "Buffkit Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: deps
deps: ## Install Go dependencies
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the buffkit package
	go build -v ./...

.PHONY: test
test: ## Run tests
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-short
test-short: ## Run short tests only
	go test -v -short ./...

.PHONY: examples
examples: ## Run the example application
	cd examples && go run main.go

.PHONY: examples-build
examples-build: ## Build the examples binary
	cd examples && go build -o ../bin/examples main.go

.PHONY: run
run: examples ## Alias for examples

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.txt
	go clean -cache

.PHONY: lint
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: check
check: fmt vet ## Run format and vet checks

.PHONY: coverage
coverage: test ## Generate and open coverage report
	go tool cover -html=coverage.txt

.PHONY: install
install: ## Install buffkit globally
	go install .

.PHONY: docker-redis
docker-redis: ## Start Redis in Docker for development
	docker run -d --name buffkit-redis -p 6379:6379 redis:alpine || docker start buffkit-redis

.PHONY: docker-mailhog
docker-mailhog: ## Start MailHog in Docker for email testing
	docker run -d --name buffkit-mailhog -p 1025:1025 -p 8025:8025 mailhog/mailhog || docker start buffkit-mailhog
	@echo "MailHog UI: http://localhost:8025"

.PHONY: docker-services
docker-services: docker-redis docker-mailhog ## Start all Docker services

.PHONY: docker-stop
docker-stop: ## Stop all Docker services
	-docker stop buffkit-redis buffkit-mailhog

.PHONY: docker-clean
docker-clean: docker-stop ## Remove all Docker containers
	-docker rm buffkit-redis buffkit-mailhog

.PHONY: watch
watch: ## Watch for changes and rebuild
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air -c .air.toml

.PHONY: dev
dev: docker-services ## Start development environment with services
	@echo "Starting development environment..."
	@echo "Redis: localhost:6379"
	@echo "MailHog SMTP: localhost:1025"
	@echo "MailHog UI: http://localhost:8025"
	@echo ""
	$(MAKE) examples

.PHONY: setup
setup: deps docker-services ## Initial project setup
	@echo "✅ Dependencies installed"
	@echo "✅ Docker services started"
	@echo ""
	@echo "Ready to run: make examples"

# Variables
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin
