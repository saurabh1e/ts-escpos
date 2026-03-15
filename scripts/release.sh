#!/bin/bash

# Check if version argument is provided
if [ -z "$1" ]; then
  echo "Error: Version argument is required."
  echo "Usage: ./scripts/release.sh v1.0.0"
  exit 1
fi

VERSION=$1

# Ensure we are on the main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ] && [ "$CURRENT_BRANCH" != "master" ]; then
  echo "Warning: You are not on 'main' or 'master' branch. Current branch: $CURRENT_BRANCH"
  read -p "Continue anyway? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

# Ensure working directory is clean
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: Working directory is not clean. Please commit or stash changes first."
  exit 1
fi

# Cleanup legacy tools directory if it exists to prevent build interference
# The presence of Go source code in build/tools causes Wails/Go to try and compile it
if [ -d "build/tools" ]; then
    echo "🧹 Cleaning up legacy build tools directory..."
    rm -rf build/tools
fi

setup_go120() {
    GO_VERSION="1.20.14"
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    # Map architecture names
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi
    if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi

    # Create distinct directory for tools
    PROJECT_ROOT=$(pwd)
    TOOLS_DIR="$PROJECT_ROOT/build/.tools"
    GO_DIR="$TOOLS_DIR/go$GO_VERSION"
    GO_BIN="$GO_DIR/bin/go"

    if [ -x "$GO_BIN" ]; then
        echo "✅ Go $GO_VERSION found at $GO_BIN"
        export GO120_CMD="$GO_BIN"
        export GO120_ROOT="$GO_DIR"
        return 0
    fi

    echo "⬇️ Downloading Go $GO_VERSION for $OS/$ARCH..."
    mkdir -p "$TOOLS_DIR"
    mkdir -p "$GO_DIR"

    ARCHIVE="go$GO_VERSION.tar.gz"
    URL="https://go.dev/dl/go$GO_VERSION.$OS-$ARCH.tar.gz"

    echo "   URL: $URL"
    if ! curl -L "$URL" -o "$ARCHIVE"; then
        echo "❌ Failed to download Go $GO_VERSION"
        exit 1
    fi

    echo "📦 Extracting Go $GO_VERSION..."
    if ! tar -xzf "$ARCHIVE" --strip-components=1 -C "$GO_DIR"; then
        echo "❌ Failed to extract Go $GO_VERSION"
        rm "$ARCHIVE"
        exit 1
    fi

    rm "$ARCHIVE"

    if [ -x "$GO_BIN" ]; then
        echo "✅ Go $GO_VERSION ready."
        export GO120_CMD="$GO_BIN"
        export GO120_ROOT="$GO_DIR"
    else
        echo "❌ Failed to setup Go $GO_VERSION"
        exit 1
    fi
}


echo "🚀 Starting release process for version $VERSION..."

# 1. build the windows installer locally
echo "🔨 Building Windows Installer..."
make release VERSION=$VERSION

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# 1b. Prepare Go 1.20 environment
setup_go120

# 1c. Build Lite versions with Go 1.20
echo "🔨 Preparing for Go 1.20 build (temporarily downgrading go.mod)..."
cp go.mod go.mod.bak
# Use sed to replace the go version line. Compatible with BSD/GNU sed by avoiding -i
sed 's/^go .*/go 1.20/' go.mod.bak > go.mod

echo "🔨 Building Lite versions with Go 1.20..."
# Use a dedicated cache and GOROOT to avoid conflicts with system Go (1.25)
export GOROOT="$GO120_ROOT"
export GOCACHE="$(pwd)/build/.cache-go1.20"
mkdir -p "$GOCACHE"
# Clean cache for safety
"$GO120_CMD" clean -cache

make build-lite-windows VERSION=$VERSION GO="$GO120_CMD"
BUILD_RES_AMD64=$?

make build-lite-windows-386 VERSION=$VERSION GO="$GO120_CMD"
BUILD_RES_386=$?

# Restore go.mod immediately
mv go.mod.bak go.mod

# Unset environment variables to avoid affecting subsequent steps
unset GOROOT
unset GOCACHE

if [ $BUILD_RES_AMD64 -ne 0 ]; then
    echo "❌ Lite (amd64) Build failed!"
    exit 1
fi

if [ $BUILD_RES_386 -ne 0 ]; then
    echo "❌ Lite (386) Build failed!"
    exit 1
fi

# Manual build of 32-bit installer
echo "🔨 Building 32-bit Installer manually..."
# Running from project root, so change directory to installer folder
pushd build/windows/installer > /dev/null

# Clean version string (remove leading 'v' if present) for NSIS
CLEAN_VERSION="${VERSION#v}"

# Pass version to makensis
makensis -DINFO_PRODUCTVERSION=$CLEAN_VERSION project_386.nsi
if [ $? -ne 0 ]; then
    echo "❌ 32-bit Installer Build failed!"
    exit 1
fi
popd > /dev/null

# Ensure compatibility by copying amd64 exe to default exe name if it doesn't exist
if [ -f "build/bin/ts-escpos-amd64.exe" ]; then
    cp "build/bin/ts-escpos-amd64.exe" "build/bin/ts-escpos.exe"
fi

# 1d. Build Lite Installers (NSIS)
echo "🔨 Building Lite Installers (NSIS)..."
pushd build/windows/installer > /dev/null

# Lite AMD64 Installer
makensis -DINFO_PRODUCTVERSION=$CLEAN_VERSION \
         -DLITE_BINARY="..\..\bin\ts-escpos-lite.exe" \
         -DLITE_ARCH="amd64" \
         project_lite.nsi

if [ $? -ne 0 ]; then
    echo "❌ Lite (amd64) Installer Build failed!"
    exit 1
fi

# Lite 386 Installer
makensis -DINFO_PRODUCTVERSION=$CLEAN_VERSION \
         -DLITE_BINARY="..\..\bin\ts-escpos-lite-32.exe" \
         -DLITE_ARCH="386" \
         project_lite.nsi

if [ $? -ne 0 ]; then
    echo "❌ Lite (386) Installer Build failed!"
    exit 1
fi

popd > /dev/null

# 2. Add binaries to git
echo "📦 Committing artifacts..."
git add -f build/bin/ts-escpos-amd64-installer.exe
git add -f build/bin/ts-escpos.exe
git add -f build/bin/ts-escpos-amd64.exe
git add -f build/bin/ts-escpos-386.exe
git add -f build/bin/ts-escpos-386-installer.exe
git add -f build/bin/ts-escpos-lite.exe
git add -f build/bin/ts-escpos-lite-32.exe
git add -f build/bin/ts-escpos-lite-amd64-installer.exe
git add -f build/bin/ts-escpos-lite-386-installer.exe

git commit -m "chore: release artifacts for $VERSION"

# 3. Create and push tag
echo "🏷️ Tagging version $VERSION..."
git tag $VERSION
git push origin $VERSION

echo "🏷️ Tagging version latest..."
git tag -f latest
git push -f origin latest

echo "✅ Done! GitHub Action should now trigger and create a release with the uploaded artifacts."
