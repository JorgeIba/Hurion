# Variables
BINARY_NAME=hurion
BUILD_DIR=./bin
SCRIPT_PATH?=app/script/test.jsonl
PORT?=8080
FLAGS?=
TEST_FLAGS?=-race

# .PHONY tells make that these names are not files, but commands
.PHONY: all build build-web test test-unit test-integration test-workflow test-web generate clean run run-dev run-mock run-server stop-server

# By default, running 'make' executes build
all: build

## generate: Run mockgen to create necessary mocks
generate:
	@echo "==> Generating mocks..."
	@go generate ./...
	@echo "==> Mocks generated successfully."

## test-unit: Run application/unit tests (depends on generate)
test-unit: generate
	@echo "==> Running unit tests (app)..."
	@go test -v $(TEST_FLAGS) ./app/...

## test-integration: Run integration tests (depends on generate)
test-integration: generate
	@echo "==> Running integration tests..."
	@go test -v $(TEST_FLAGS) ./integration/...

## test-workflow: Run workflow tests (depends on generate)
test-workflow: generate
	@echo "==> Running workflow tests..."
	@go test -v $(TEST_FLAGS) ./workflow/...

## test-web: Run web server tests (depends on generate)
test-web: generate
	@echo "==> Running web server tests..."
	@go test -v $(TEST_FLAGS) ./web/...

## test: Generate mocks and run unit, integration, workflow, and web test suites
test: test-unit test-integration test-workflow test-web
	@echo "==> All tests completed successfully."

## coverage: Run tests with coverage and generate HTML report
coverage: generate
	@echo "==> Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "==> Coverage report generated at coverage.html"

## build: Compile the bot and save binary to ./bin
build:
	@echo "==> Formatting code..."
	@go fmt ./...
	@echo "==> Compiling binary..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) app/main.go
	@echo "==> Done! Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

## run: Build and execute the bot (compose flags via FLAGS="--seed --exporter")
run: build
	@echo "==> Starting bot..."
	@$(BUILD_DIR)/$(BINARY_NAME) $(FLAGS)

## run-dev: Full local development mode (--seed --exporter)
run-dev: build
	@echo "==> Starting bot in Dev Mode (--seed --exporter)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --seed --exporter $(FLAGS)

## run-mock: Offline script testing mode (override with SCRIPT_PATH=...)
run-mock: build
	@echo "==> Running bot in Mock Script Mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) --mock-discord --script $(SCRIPT_PATH) $(FLAGS)

## build-web: Compile React frontend assets into web/dist
build-web:
	@echo "==> Compiling Web UI assets (pnpm build)..."
	@pnpm --dir web build
	@echo "==> Web UI assets compiled successfully."

## run-server: Build frontend assets and launch local Web UI dev server (customizable via PORT=9090 or FLAGS="-port 9090")
run-server: build-web
	@echo "==> Starting local Web UI dev server on port $(PORT)..."
	@PORT=$(PORT) go run web/cmd/main.go $(FLAGS)

## stop-server: Stop any running Web UI server using PID tracking with kill -0 validation
stop-server:
	@echo "==> Stopping Web UI server..."
	@if [ -f web/.server.pid ]; then \
		PID=$$(cat web/.server.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			echo "==> Gracefully terminating Web UI server (PID $$PID)..."; \
			kill -15 $$PID 2>/dev/null || true; \
		else \
			echo "==> Stale PID file found ($$PID). Process is no longer running."; \
		fi; \
		rm -f web/.server.pid; \
	else \
		fuser -k $(PORT)/tcp 2>/dev/null || echo "No server running on port $(PORT)."; \
	fi

## clean: Remove binary and temporary files
clean:
	@echo "==> Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "==> Folder $(BUILD_DIR) removed."
