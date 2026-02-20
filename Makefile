# FenixCRM Makefile
# Task 1.1: Project Setup
# Following implementation plan exactly

.PHONY: all test build run lint fmt complexity pattern-refactor-gate pattern-opportunities-gate race-stability coverage-gate coverage-app coverage-app-gate coverage-tdd check migrate-up migrate-down migrate-create migrate-version sqlc-generate docker-build docker-run e2e clean db-shell doorstop-check trace-check contract-test trace-report

# Variables
BINARY_NAME=fenix
MODULE=github.com/matiasleandrokruk/fenix
BUILD_DIR=.
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Default target
all: test build

# Run all tests (unit + integration)
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run only unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

# Run only integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -run Integration ./...

# Run E2E tests (requires UI built)
test-e2e:
	@echo "Running E2E tests..."
	cd tests/e2e && npm test

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/fenix

# Build for multiple platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/fenix
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/fenix
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/fenix
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/fenix

# Run the server (dev mode)
run:
	@echo "Running server..."
	go run ./cmd/fenix

# Run with hot reload (requires air)
dev:
	@echo "Running with hot reload..."
	air -c .air.toml

# Run linter
GOLANGCI_LINT=$(shell go env GOPATH)/bin/golangci-lint
lint:
	@echo "Running linter..."
	@test -f $(GOLANGCI_LINT) || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOLANGCI_LINT) run ./...

# Check cyclomatic complexity — fails if any function exceeds threshold 7
# Threshold rationale: 7 is the industry-standard limit for maintainable code.
# Functions above 7 are harder to test (need more test cases) and harder to understand.
# Usage: make complexity
COMPLEXITY_THRESHOLD=7
GOCYCLO=$(shell go env GOPATH)/bin/gocyclo
# Find production Go files (exclude *_test.go) under internal/
PROD_GO_FILES=$(shell find ./internal -name '*.go' ! -name '*_test.go')
complexity:
	@echo "Checking cyclomatic complexity (threshold: $(COMPLEXITY_THRESHOLD), production code only)..."
	@test -f $(GOCYCLO) || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@$(GOCYCLO) -over $(COMPLEXITY_THRESHOLD) -avg $(PROD_GO_FILES) 2>&1 | \
		tee /tmp/gocyclo_out.txt; \
	violations=$$(grep -v "^Average:" /tmp/gocyclo_out.txt); \
	if [ -n "$$violations" ]; then \
		echo "FAILED: functions above threshold $(COMPLEXITY_THRESHOLD) found — refactor before merging"; \
		exit 1; \
	else \
		echo "PASSED: all production functions at or below $(COMPLEXITY_THRESHOLD)"; \
	fi

# Run all quality gates before merge (pattern gate in warn mode + complexity + lint + tests)
check: pattern-refactor-gate complexity lint test
	@echo "All quality gates passed — safe to merge."

# Pattern refactor evidence gate (MVP):
# - warn mode: reports findings without failing
# - strict mode: fails when no/invalid evidence or strong smells are detected
PATTERN_GATE_MODE?=warn
PATTERN_GATE_TS_DUP_THRESHOLD?=2
pattern-refactor-gate:
	@echo "Running pattern refactor gate (mode: $(PATTERN_GATE_MODE))..."
	@bash ./scripts/pattern-refactor-gate.sh \
		--mode "$(PATTERN_GATE_MODE)" \
		--root . \
		--ts-dup-threshold "$(PATTERN_GATE_TS_DUP_THRESHOLD)"

# Alias semántico para oportunidades de refactor con patrones (dupl + jscpd + evidencia)
pattern-opportunities-gate: pattern-refactor-gate

# Re-run race-sensitive test package multiple times to catch flaky data races
RACE_STABILITY_COUNT?=3
race-stability:
	@echo "Running race stability checks (count: $(RACE_STABILITY_COUNT))..."
	go test -race -count=$(RACE_STABILITY_COUNT) ./internal/api/handlers

# Coverage gate over app-relevant profile generated from `coverage.out`.
# Excludes generated/sqlc and bootstrap wiring to avoid penalizing quality gate
# with non-business code that is intentionally not unit tested.
# Task 2.6 baby-steps phase 3 target.
COVERAGE_MIN?=79
coverage-gate:
	@echo "Checking global coverage threshold ($(COVERAGE_MIN)%)..."
	@awk 'NR==1{print; next} \
		$$0 !~ /internal\/infra\/sqlite\/sqlcgen\// && \
		$$0 !~ /cmd\/fenix\/main.go/ && \
		$$0 !~ /cmd\/frtrace\/main.go/ && \
		$$0 !~ /internal\/version\// && \
		$$0 !~ /ruleguard\// {print}' coverage.out > coverage_gate.out; \
	total=$$(go tool cover -func=coverage_gate.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	if [ -z "$$total" ]; then \
		echo "FAILED: could not read total coverage from coverage_gate.out"; \
		exit 1; \
	fi; \
	echo "Total coverage (gate scope): $$total%"; \
	awk -v cov="$$total" -v min="$(COVERAGE_MIN)" 'BEGIN { if (cov+0 < min+0) { exit 1 } }' || { \
		echo "FAILED: total coverage $$total% is below threshold $(COVERAGE_MIN)%"; \
		exit 1; \
	}; \
	echo "PASSED: coverage gate met"

# Coverage over application code (excluding generated/sqlc and bootstrap wiring)
# Baby step phase 3 target towards 80%.
COVERAGE_APP_MIN?=79
coverage-app:
	@echo "Generating app-only coverage profile (excluding generated/bootstrap code)..."
	@awk 'NR==1{print; next} \
		$$0 !~ /internal\/infra\/sqlite\/sqlcgen\// && \
		$$0 !~ /cmd\/fenix\/main.go/ && \
		$$0 !~ /cmd\/frtrace\/main.go/ && \
		$$0 !~ /internal\/version\// && \
		$$0 !~ /ruleguard\// {print}' coverage.out > coverage_app.out
	@go tool cover -func=coverage_app.out | tail -n 1

coverage-app-gate: coverage-app
	@echo "Checking app coverage threshold ($(COVERAGE_APP_MIN)%)..."
	@total=$$(go tool cover -func=coverage_app.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	if [ -z "$$total" ]; then \
		echo "FAILED: could not read total coverage from coverage_app.out"; \
		exit 1; \
	fi; \
	echo "App coverage: $$total%"; \
	awk -v cov="$$total" -v min="$(COVERAGE_APP_MIN)" 'BEGIN { if (cov+0 < min+0) { exit 1 } }' || { \
		echo "FAILED: app coverage $$total% is below threshold $(COVERAGE_APP_MIN)%"; \
		exit 1; \
	}; \
	echo "PASSED: app coverage gate met"

# Additional gate focused on TDD-heavy packages
# Baby step phase 3 target towards 80%.
TDD_COVERAGE_MIN?=79
coverage-tdd:
	@echo "Checking TDD package coverage threshold ($(TDD_COVERAGE_MIN)%)..."
	go test -coverprofile=coverage_tdd.out ./internal/api ./internal/api/handlers ./internal/domain/knowledge ./pkg/auth >/tmp/tdd_coverage.log 2>&1
	@total=$$(go tool cover -func=coverage_tdd.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	if [ -z "$$total" ]; then \
		echo "FAILED: could not read total coverage from coverage_tdd.out"; \
		cat /tmp/tdd_coverage.log; \
		exit 1; \
	fi; \
	echo "TDD coverage: $$total%"; \
	awk -v cov="$$total" -v min="$(TDD_COVERAGE_MIN)" 'BEGIN { if (cov+0 < min+0) { exit 1 } }' || { \
		echo "FAILED: TDD coverage $$total% is below threshold $(TDD_COVERAGE_MIN)%"; \
		exit 1; \
	}; \
	echo "PASSED: TDD coverage gate met"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Apply pending migrations
migrate-up:
	@echo "Applying migrations..."
	go run ./cmd/fenix migrate up

# Rollback last migration
migrate-down:
	@echo "Rolling back migration..."
	go run ./cmd/fenix migrate down

# Show current migration version
# Task 1.2.9: DB targets added
migrate-version:
	@echo "Checking migration version..."
	go run ./cmd/fenix migrate version

# Open SQLite shell (requires sqlite3 CLI)
db-shell:
	@echo "Opening SQLite shell..."
	sqlite3 ./data/fenixcrm.db

# Create new migration
	@echo "Creating migration $(NAME)..."
	@read -p "Enter migration name: " name; \
	go run ./cmd/fenix migrate create $$name

# Generate Go code from SQL queries
sqlc-generate:
	@echo "Generating sqlc code..."
	sqlc generate

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -v ./data:/data -p 8080:8080 $(BINARY_NAME):latest

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out
	rm -f coverage_app.out

# Install development dependencies
install-tools:
	@echo "Installing dev tools..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "Note: gocognit + maintidx are bundled with golangci-lint (no separate install needed)"

# CI target - runs all checks (complexity/race/coverage gates before build)
doorstop-check:
	@echo "Checking Doorstop requirement integrity..."
	@./.venv/bin/doorstop -j ./reqs -L -R -W

trace-check:
	@echo "Checking FR-to-test traceability..."
	@go run ./cmd/frtrace -reqs ./reqs -root .

contract-test: build
	@echo "Running API contract tests..."
	@bash tests/contract/run.sh

trace-report:
	@./.venv/bin/doorstop publish all ./docs/trace-report

ci: fmt complexity pattern-refactor-gate doorstop-check trace-check lint test race-stability coverage-gate coverage-app-gate coverage-tdd build contract-test
	@echo "All CI checks passed!"

# Version info
version:
	@echo "Version: $(VERSION)"
	@./$(BINARY_NAME) --version 2>/dev/null || echo "Binary not built yet"

# Help
help:
	@echo "FenixCRM Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make test              - Run all tests"
	@echo "  make test-unit         - Run unit tests only"
	@echo "  make test-integration  - Run integration tests only"
	@echo "  make test-e2e          - Run E2E tests"
	@echo "  make build             - Build binary"
	@echo "  make build-all         - Build for all platforms"
	@echo "  make run               - Run server (dev)"
	@echo "  make dev               - Run with hot reload"
	@echo "  make lint              - Run linter"
	@echo "  make complexity        - Check cyclomatic complexity (threshold: 7)"
	@echo "  make pattern-refactor-gate - Check evidence/signals for design-pattern refactors"
	@echo "  make pattern-opportunities-gate - Alias for pattern-refactor-gate"
	@echo "  make race-stability    - Run race checks repeatedly on handler tests"
	@echo "  make coverage-gate     - Enforce global coverage threshold"
	@echo "  make coverage-app-gate - Enforce app-only coverage threshold"
	@echo "  make coverage-tdd      - Enforce TDD package coverage threshold"
	@echo "  make doorstop-check    - Validate Doorstop requirement integrity"
	@echo "  make trace-check       - Check FR-to-test traceability (Go scanner)"
	@echo "  make contract-test     - Run API contract tests (Schemathesis)"
	@echo "  make trace-report      - Publish traceability HTML report"
	@echo "  make check             - Run all quality gates (complexity + lint + tests)"
	@echo "  make fmt               - Format code"
	@echo "  make migrate-up        - Apply migrations"
	@echo "  make migrate-down      - Rollback migration"
	@echo "  make migrate-create    - Create new migration"
	@echo "  make sqlc-generate     - Generate sqlc code"
	@echo "  make docker-build      - Build Docker image"
	@echo "  make docker-run        - Run Docker container"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make install-tools     - Install dev dependencies"
	@echo "  make ci                - Run all CI checks"
	@echo "  make version           - Show version info"
	@echo "  make help              - Show this help"
