#!/bin/bash
# Build script for bot-service

set -e

echo "Building bot-service for current platform..."
go build -o bot-service cmd/bot-service/main.go

echo ""
echo "✅ Build complete!"
echo ""
ls -lh bot-service
echo ""
echo "File info:"
file bot-service
