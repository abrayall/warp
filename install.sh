#!/bin/bash
set -e

echo "Installing warp..."

# Build
go build -o warp ./framework/cli

# Install to GOPATH/bin or /usr/local/bin
INSTALL_DIR="${GOPATH:-$HOME/go}/bin"
if [ ! -d "$INSTALL_DIR" ]; then
    INSTALL_DIR="/usr/local/bin"
fi

cp warp "$INSTALL_DIR/warp"
rm -f warp

echo "Installed warp to ${INSTALL_DIR}/warp"
echo "Run 'warp --help' to get started"
