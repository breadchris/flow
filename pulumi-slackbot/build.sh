#!/bin/bash

# Build script for Slackbot Lambda deployment

set -e

echo "Building Slackbot Lambda functions..."

# Create build directory
mkdir -p build

# Build Slackbot Lambda
echo "Building Slackbot Lambda..."
cd lambda
GOOS=linux GOARCH=amd64 go build -o ../build/slackbot-lambda main.go
cd ..

# Build Claude Session Lambda
echo "Building Claude Session Lambda..."
cd lambda/claude-session
GOOS=linux GOARCH=amd64 go build -o ../../build/claude-session-lambda main.go
cd ../..

# Create deployment packages
echo "Creating deployment packages..."

# Slackbot Lambda package
cd build
zip -r slackbot-lambda.zip slackbot-lambda
cd ..

# Claude Session Lambda package
cd build
zip -r claude-session-lambda.zip claude-session-lambda
cd ..

# Copy deployment packages to lambda directory
cp build/slackbot-lambda.zip lambda/
cp build/claude-session-lambda.zip lambda/

echo "Build complete!"
echo "Lambda packages:"
echo "  - lambda/slackbot-lambda.zip"
echo "  - lambda/claude-session-lambda.zip"