#!/bin/bash

OUT_DIR="release"
rm -rf $OUT_DIR
mkdir -p $OUT_DIR

VERSION="1.2.1-cli"

echo "Building version $VERSION..."

# Generate metadata.json
echo "{\"version\": \"$VERSION\"}" > $OUT_DIR/metadata.json

echo "Building for macOS (AMD64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $OUT_DIR/temp_amd64 main.go

# Package macOS AMD64 App
APP_DIR="$OUT_DIR/HealthAgent-Mac-Intel.app"
mkdir -p "$APP_DIR/Contents/MacOS"
cp Info.plist "$APP_DIR/Contents/"
cp $OUT_DIR/temp_amd64 "$APP_DIR/Contents/MacOS/health-agent"
chmod +x "$APP_DIR/Contents/MacOS/health-agent"
# Ad-hoc sign the app to prevent "Damaged" error
codesign --force --deep -s - "$APP_DIR"
# Zip the app
cd $OUT_DIR
zip -r agent-$VERSION-mac-amd64.zip HealthAgent-Mac-Intel.app
rm -rf HealthAgent-Mac-Intel.app temp_amd64
cd ..


echo "Building for macOS (ARM64/Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o $OUT_DIR/temp_arm64 main.go

# Package macOS ARM64 App
APP_DIR="$OUT_DIR/HealthAgent-Mac-Silicon.app"
mkdir -p "$APP_DIR/Contents/MacOS"
cp Info.plist "$APP_DIR/Contents/"
cp $OUT_DIR/temp_arm64 "$APP_DIR/Contents/MacOS/health-agent"
chmod +x "$APP_DIR/Contents/MacOS/health-agent"
# Ad-hoc sign the app to prevent "Damaged" error
codesign --force --deep -s - "$APP_DIR"
# Zip the app
cd $OUT_DIR
zip -r agent-$VERSION-mac-arm64.zip HealthAgent-Mac-Silicon.app
rm -rf HealthAgent-Mac-Silicon.app temp_arm64
cd ..


echo "Building for Linux (AMD64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $OUT_DIR/agent-linux-amd64 main.go
cd $OUT_DIR
zip agent-$VERSION-linux-amd64.zip agent-linux-amd64
rm agent-linux-amd64
cd ..

echo "Building for Windows (AMD64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $OUT_DIR/agent-windows-amd64.exe main.go
cd $OUT_DIR
zip agent-$VERSION-windows-amd64.zip agent-windows-amd64.exe
rm agent-windows-amd64.exe
cd ..

echo "Build complete. Versioned zips and metadata.json created in 'release/'."

# Auto-deploy to both UM and Root-Config (Shell)
TARGET_UM="../um/public/agent"
TARGET_SHELL="../root-config/public/agent"

mkdir -p $TARGET_UM
mkdir -p $TARGET_SHELL
cp $OUT_DIR/* $TARGET_UM/
cp $OUT_DIR/* $TARGET_SHELL/

echo "Deployed artifacts to $TARGET_UM and $TARGET_SHELL"
