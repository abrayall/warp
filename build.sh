#!/bin/bash
set -e

# Get version using vermouth
VERSION=$(vermouth 2>/dev/null || curl -sfL https://raw.githubusercontent.com/abrayall/vermouth/refs/heads/main/vermouth.sh | sh -)

LDFLAGS="-X warp/framework/cli/cmd.Version=${VERSION}"

echo "Building warp ${VERSION}..."

# Build for current platform
go build -ldflags "${LDFLAGS}" -o warp ./framework/cli

echo "Built: ./warp"

# Cross-compile if requested
if [ "$1" = "--all" ]; then
    mkdir -p dist

    GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o dist/warp-darwin-amd64 ./framework/cli
    GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o dist/warp-darwin-arm64 ./framework/cli
    GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o dist/warp-linux-amd64 ./framework/cli
    GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o dist/warp-linux-arm64 ./framework/cli

    echo "Cross-compiled binaries in dist/"
fi
