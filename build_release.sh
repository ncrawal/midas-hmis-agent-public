#!/bin/bash

# Normalized Build Script for Health HMIS Agent
# Handles macOS (native), Linux (Docker), Windows (Docker)

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Extract Version from Go source
VERSION=$(grep 'AgentVersion =' internal/models/models.go | cut -d '"' -f 2)
APP_NAME="health-hmis-agent"

echo -e "${BLUE}🔨 Health HMIS Agent Build System v${VERSION}${NC}"
echo "=============================="

mkdir -p release

# Function: Build macOS (runs on host Mac)
build_macos() {
    echo -e "\n${BLUE}🍎 Building for macOS...${NC}"
    rm -rf build/bin
    
    # Build Universal Binary inside .app
    wails build -platform darwin/universal -clean
    
    # Find the generated .app (Wails uses 'name' from wails.json, not 'outputfilename')
    APP_BUNDLE=$(find build/bin -maxdepth 1 -name "*.app" | head -n 1)
    
    if [ -n "$APP_BUNDLE" ]; then
        echo "✅ Found app bundle: $APP_BUNDLE"
        DMG_NAME="release/HealthHMISAgent-v${VERSION}-macOS.dmg"
        rm -f "$DMG_NAME"
        
        # Create temp folder for DMG content
        mkdir -p build/dmg_content
        cp -r "$APP_BUNDLE" build/dmg_content/
        
        # Create DMG
        hdiutil create -volname "Health HMIS Agent" -srcfolder "build/dmg_content" -ov -format UDZO "$DMG_NAME"
        rm -rf build/dmg_content
        
        echo -e "${GREEN}✓ macOS DMG created: $DMG_NAME${NC}"
    else
        echo -e "${RED}❌ macOS build failed (.app not found)${NC}"
        exit 1
    fi
}

# Function: Build Linux & Windows (runs inside Docker)
build_linux_windows() {
    echo -e "\n${BLUE}🐧/🪟 Building for Linux & Windows (inside container)...${NC}"
    
    # -----------------------------------------------------
    # 1. Linux Binary & DEB
    # -----------------------------------------------------
    echo "This building process will take some time..."
    echo "   Building Linux binary..."
    wails build -platform linux/amd64 -clean -o $APP_NAME
    
    LINUX_BIN="build/bin/${APP_NAME}"
    
    if [ -f "$LINUX_BIN" ]; then
        # Create DEB package structure manually
        echo "   Pkg: Building .deb..."
        DEB_DIR="build/deb_pkg"
        rm -rf "$DEB_DIR"
        mkdir -p "$DEB_DIR/DEBIAN"
        mkdir -p "$DEB_DIR/usr/local/bin"
        mkdir -p "$DEB_DIR/usr/share/applications"
        
        # Binary
        cp "$LINUX_BIN" "$DEB_DIR/usr/local/bin/$APP_NAME"
        chmod +x "$DEB_DIR/usr/local/bin/$APP_NAME"
        
        # Control File
        cat > "$DEB_DIR/DEBIAN/control" << EOF
Package: health-hmis-agent
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Health HMIS Team
Depends: libgtk-3-0, libwebkit2gtk-4.0-37 | libwebkit2gtk-4.1-0
Description: Health HMIS Agent for silent printing and device integration.
EOF

        # Desktop Entry
        cat > "$DEB_DIR/usr/share/applications/$APP_NAME.desktop" << EOF
[Desktop Entry]
Name=Health HMIS Agent
Comment=Silent Printing Agent
Exec=/usr/local/bin/$APP_NAME
Type=Application
Terminal=false
Categories=Utility;
EOF

        # Build DEB
        dpkg-deb --build "$DEB_DIR" "release/HealthHMISAgent-v${VERSION}-amd64.deb"
        echo -e "${GREEN}   ✓ DEB created: release/HealthHMISAgent-v${VERSION}-amd64.deb${NC}"
        
        # Also zip the binary for non-Debian users
        zip -j "release/HealthHMISAgent-v${VERSION}-linux-amd64.zip" "$LINUX_BIN"
        echo -e "${GREEN}   ✓ Linux Zip created${NC}"
    else
        echo -e "${RED}❌ Linux build failed${NC}"
    fi

    # -----------------------------------------------------
    # 2. Windows EXE & Installer
    # -----------------------------------------------------
    echo "   Building Windows binary..."
    # Attempt to build with NSIS installer if supported
    wails build -platform windows/amd64 -clean -o "${APP_NAME}.exe" 
    
    WIN_EXE="build/bin/${APP_NAME}.exe"
    
    if [ -f "$WIN_EXE" ]; then
        # Rename and zip portable exe
        cp "$WIN_EXE" "release/HealthHMISAgent-v${VERSION}.exe"
        zip -j "release/HealthHMISAgent-v${VERSION}-windows-amd64.zip" "$WIN_EXE"
        echo -e "${GREEN}   ✓ Windows .exe and .zip created${NC}"
        
        # Check if installer was generated (needs wails nsis config)
        # If wails.json doesn't configure nsis, this file won't exist.
        # But we can try 'wails build -nsis'
    else
        echo -e "${RED}❌ Windows build failed${NC}"
    fi
}


# MAIN LOGIC
# Check if running inside Docker (simple check for /.dockerenv or hostname)
if [ "$1" == "linux-docker" ]; then
    build_linux_windows
    
    # Fix permissions for artifacts created inside docker (owned by root usually)
    # chown -R $(stat -c "%u:%g" .) release
    # Above simpler: chmod 777 release/*
    chmod -R 777 release
    
    exit 0
fi

# Detect OS Logic
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Running on Host Mac
    
    # 1. Build macOS Native
    build_macos
    
    # 2. Ask for Linux/Windows Docker Build
    echo -e "\n${YELLOW}❓ Do you want to build Linux (Deb/Zip) & Windows (Exe) using Docker? (y/n)${NC}"
    read -p "> " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if ! command -v docker >/dev/null 2>&1; then
            echo -e "${RED}❌ Docker not found. Install Docker Desktop to build for Linux/Windows.${NC}"
            exit 1
        fi
        
        echo -e "${BLUE}🐳 Building Docker Image 'health-agent-builder'...${NC}"
        docker build -t health-agent-builder -f Dockerfile.linux .
        
        echo -e "${BLUE}🐳 Running Build inside Docker...${NC}"
        # Mount current dir to /app
        docker run --rm -v "$(pwd):/app" health-agent-builder
    else
        echo "Skipping Linux/Windows build."
    fi
    
else
    # Assume running on Linux host natively
    build_linux_windows
fi

echo -e "\n${GREEN}🎉 Build process finished! Check the 'release' folder.${NC}"
ls -lh release/
