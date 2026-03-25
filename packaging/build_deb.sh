#!/bin/bash
set -e

# Configuration
PKG_DIR="packaging/deb"
BIN_NAME="termviewer-agent"
DEB_NAME="termviewer-agent_1.0.0_amd64.deb"

echo "Building TermViewer Agent for amd64..."
cd agent
go build -o ../$PKG_DIR/usr/bin/$BIN_NAME main.go
cd ..

# Ensure permissions are correct
chmod 755 $PKG_DIR/DEBIAN
chmod 755 $PKG_DIR/usr/bin/$BIN_NAME
chmod 644 $PKG_DIR/lib/systemd/system/termviewer-agent.service

# Create necessary directories
mkdir -p $PKG_DIR/var/lib/termviewer

echo "Creating Debian package..."
dpkg-deb --build $PKG_DIR $DEB_NAME

echo "Successfully built $DEB_NAME"
