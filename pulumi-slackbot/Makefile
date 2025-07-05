# Makefile for Slackbot Lambda deployment

.PHONY: build deploy clean test help

# Default target
help:
	@echo "Available targets:"
	@echo "  build     - Build Lambda functions"
	@echo "  deploy    - Deploy infrastructure with Pulumi"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  init      - Initialize Pulumi stack"
	@echo "  config    - Configure Pulumi settings"
	@echo "  destroy   - Destroy infrastructure"
	@echo "  logs      - View Lambda logs"
	@echo "  help      - Show this help message"

# Build Lambda functions
build:
	@echo "Building Lambda functions..."
	@chmod +x build.sh
	@./build.sh

# Deploy infrastructure
deploy: build
	@echo "Deploying infrastructure..."
	@pulumi up --yes

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/
	@rm -f lambda/*.zip
	@rm -f lambda/claude-session/*.zip

# Run tests
test:
	@echo "Running tests..."
	@cd lambda && go test -v ./...
	@cd lambda/claude-session && go test -v ./...

# Initialize Pulumi stack
init:
	@echo "Initializing Pulumi stack..."
	@pulumi stack init dev

# Configure Pulumi settings
config:
	@echo "Configuring Pulumi settings..."
	@pulumi config set aws:region us-east-1
	@echo "Please set the following secrets manually:"
	@echo "  pulumi config set slackbot:slackBotToken 'xoxb-your-bot-token' --secret"
	@echo "  pulumi config set slackbot:slackSigningSecret 'your-signing-secret' --secret"
	@echo "  pulumi config set slackbot:claudeApiKey 'your-claude-api-key' --secret"

# Destroy infrastructure
destroy:
	@echo "Destroying infrastructure..."
	@pulumi destroy --yes

# View Lambda logs
logs:
	@echo "Viewing Lambda logs..."
	@aws logs tail /aws/lambda/slackbot-lambda --follow

# Development targets
dev-build:
	@echo "Building for development..."
	@cd lambda && go build -o slackbot-lambda main.go
	@cd lambda/claude-session && go build -o claude-session-lambda main.go

dev-test:
	@echo "Running development tests..."
	@cd lambda && go run main.go

# Format code
fmt:
	@echo "Formatting code..."
	@cd lambda && go fmt ./...
	@cd lambda/claude-session && go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@cd lambda && go vet ./...
	@cd lambda/claude-session && go vet ./...

# Security scan
security:
	@echo "Running security scan..."
	@cd lambda && go list -json -m all | nancy sleuth
	@cd lambda/claude-session && go list -json -m all | nancy sleuth

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	@cd lambda && go get -u ./... && go mod tidy
	@cd lambda/claude-session && go get -u ./... && go mod tidy

# Get deployment info
info:
	@echo "Getting deployment info..."
	@pulumi stack output

# Get stack status
status:
	@echo "Getting stack status..."
	@pulumi stack ls

# Preview changes
preview:
	@echo "Previewing changes..."
	@pulumi preview

# Export stack
export:
	@echo "Exporting stack..."
	@pulumi stack export --file stack-export.json

# Import stack
import:
	@echo "Importing stack..."
	@pulumi stack import --file stack-export.json