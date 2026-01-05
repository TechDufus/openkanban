.PHONY: build test test-unit test-integration test-all coverage lint clean help

GO := go
BINARY := openkanban
COVERAGE_FILE := coverage.out

build:
	$(GO) build -o $(BINARY) .

test: test-unit

test-unit:
	$(GO) test -race ./...

test-integration:
	$(GO) test -race -tags integration ./...

test-all: test-unit test-integration

coverage:
	$(GO) test -race -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html

coverage-integration:
	$(GO) test -race -tags integration -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html

lint:
	$(GO) vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; fi

clean:
	rm -f $(BINARY) $(COVERAGE_FILE) coverage.html

help:
	@echo "Available targets:"
	@echo "  build             - Build the binary"
	@echo "  test              - Run unit tests (default)"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  test-all          - Run all tests (unit + integration)"
	@echo "  coverage          - Generate coverage report (unit tests)"
	@echo "  coverage-integration - Generate coverage report (all tests)"
	@echo "  lint              - Run linters"
	@echo "  clean             - Remove build artifacts"
