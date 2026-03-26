#!/bin/bash
# Build script for bot-service

set -e

# Detect current OS and architecture
OS_TYPE=$(uname -s)
ARCH_TYPE=$(uname -m)

echo "Building bot-service for current platform..."
echo "Detected: $OS_TYPE $ARCH_TYPE"
echo ""

# Determine which binary to build
if [[ "$OS_TYPE" == "Darwin" ]]; then
    if [[ "$ARCH_TYPE" == "arm64" ]]; then
        OUTPUT="bot-service-darwin-arm64"
        echo "Building for macOS ARM64 (Apple Silicon)..."
        GOOS=darwin GOARCH=arm64 go build -o "$OUTPUT" cmd/bot-service/main.go
    else
        echo "❌ Error: macOS AMD64 (Intel) is not supported."
        echo "   Please use an Apple Silicon Mac or build for Linux."
        exit 1
    fi
elif [[ "$OS_TYPE" == "Linux" ]]; then
    if [[ "$ARCH_TYPE" == "x86_64" ]]; then
        OUTPUT="bot-service-linux-amd64"
        echo "Building for Linux AMD64..."
        GOOS=linux GOARCH=amd64 go build -o "$OUTPUT" cmd/bot-service/main.go
    else
        echo "❌ Error: Unsupported architecture: $ARCH_TYPE"
        exit 1
    fi
else
    echo "❌ Error: Unsupported OS: $OS_TYPE"
    exit 1
fi

echo ""
echo "✅ Build complete!"
echo ""
ls -lh "$OUTPUT"
echo ""
echo "File info:"
file "$OUTPUT"
