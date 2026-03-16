# Variables
BINARY_NAME=huron-bot
BUILD_DIR=./bin

# .PHONY tells make that these names are not files, but commands
.PHONY: all build test generate clean

# By default, running 'make' executes build
all: build

## generate: Run mockgen to create necessary mocks
generate:
	@echo "==> Generating mocks..."
	@go generate ./...

## test: Generate mocks and then run all project tests
test: generate
	@echo "==> Running tests..."
	@go test -v ./app/...

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
