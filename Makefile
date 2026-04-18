# Variables
BINARY_NAME=huron-bot
BUILD_DIR=./bin

# .PHONY tells make that these names are not files, but commands
.PHONY: all build test test-unit test-integration generate clean

# By default, running 'make' executes build
all: build

## generate: Run mockgen to create necessary mocks
generate:
	@echo "==> Generating mocks..."
	@go generate ./...
	@echo "==> Mocks generated successfully."

## test: Generate mocks and run both unit and integration test suites
test: generate test-unit test-integration
	@echo "==> All tests completed successfully."

## test-unit: Run application/unit tests
test-unit:
	@echo "==> Running unit tests (app)..."
	@go test -v ./app/...

## test-integration: Run integration tests
test-integration:
	@echo "==> Running integration tests..."
	@go test -v ./integration/...

## coverage: Run tests with coverage and generate HTML report
coverage: generate
	@echo "==> Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "==> Coverage report generated at coverage.html"

## build: Compile the bot and save binary to ./bin
build:
	@echo "==> Compiling binary..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) app/main.go
	@echo "==> Done! Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

## run: Build and execute the bot
run: build
	@echo "==> Starting bot..."
	@$(BUILD_DIR)/$(BINARY_NAME)

## clean: Remove binary and temporary files
clean:
	@echo "==> Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "==> Folder $(BUILD_DIR) removed."
