#!/bin/bash

# Configuration
VERSION="1.3.0"
OUT_DIR="release"
APP_NAME="health-hmis-agent"

# Ensure Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running or not installed."
    exit 1
fi

echo "🐳 Building Linux (AMD64) using Docker..."
echo "This ensures we build a compatible binary for Linux systems."

# Use official Golang image as base (contains Go, we add Node + Wails deps)
docker run --rm \
  -v "$PWD":/app \
  -w /app \
  golang:1.21-bullseye \
  /bin/bash -c "
    echo '🔹 Setting up Linux build environment...'
    
    # Install system dependencies
    apt-get update >/dev/null
    apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev >/dev/null
    
    # Install Node.js (v18)
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash - >/dev/null
    apt-get install -y nodejs >/dev/null
    
    # Install Wails
    echo '🔹 Installing Wails CLI...'
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    
    # Build
    echo '🔹 Building Linux binary...'
    /go/bin/wails build -platform linux/amd64 -clean -o $APP_NAME
    
    # Check if build succeeded
    if [ -f 'build/bin/$APP_NAME' ]; then
        # Ensure proper permissions for host user
        chmod 777 build/bin/$APP_NAME
        echo '✅ Build inside Docker successful.'
    else
        echo '❌ Build inside Docker failed.'
        exit 1
    fi
  "

# Check if the binary appeared on host
if [ -f "build/bin/$APP_NAME" ]; then
    echo "✅ Linux binary created at build/bin/$APP_NAME"
    
    # Package it
    mkdir -p $OUT_DIR
    zip "$OUT_DIR/Agent-$VERSION-Linux-amd64.zip" "build/bin/$APP_NAME"
    echo "📦 Zipped to $OUT_DIR/Agent-$VERSION-Linux-amd64.zip"
else
    echo "❌ Linux build failed."
    exit 1
fi
