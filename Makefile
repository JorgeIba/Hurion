# Variables
BINARY_NAME=huron-bot
BUILD_DIR=./bin

# .PHONY tells make that these names are not files, but commands
.PHONY: all build test test-unit test-integration test-workflow generate clean run-seed

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
	@go test -v ./app/...

## test-integration: Run integration tests (depends on generate)
test-integration: generate
	@echo "==> Running integration tests..."
	@go test -v ./integration/...

## test-workflow: Run workflow tests (depends on generate)
test-workflow: generate
	@echo "==> Running workflow tests..."
	@go test -v ./workflow/...

## test: Generate mocks and run unit, integration, and workflow test suites
test: test-unit test-integration test-workflow
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

## run: Build and execute the bot
run: build
	@echo "==> Starting bot..."
	@$(BUILD_DIR)/$(BINARY_NAME)

## run-seed: Build and execute the bot with database seeding
run-seed: build
	@echo "==> Seeding database and starting bot..."
	@$(BUILD_DIR)/$(BINARY_NAME) --seed

## clean: Remove binary and temporary files
clean:
	@echo "==> Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "==> Folder $(BUILD_DIR) removed."
