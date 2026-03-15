# Makefile for ts-escpos Wails project

.PHONY: all dev build build-windows installer-windows deps clean

# Default target
all: build

# Run in development mode
dev:
	wails dev

# Build for the current platform
build:
	wails build

# Build for Windows (amd64)
build-windows:
	wails build -platform windows/amd64

# Build for Windows (386)
build-windows-386:
	wails build -platform windows/386

# Build Windows Installer (requires NSIS installed)
# On macOS: brew install nsis
installer-windows:
	wails build -platform windows/amd64 -nsis

# Install dependencies (Go and Frontend)
deps:
	go mod tidy
	cd frontend && pnpm install

# Clean build artifacts
clean:
	rm -rf build/bin

# Build release (Windows Installer)
# Usage: make release VERSION=v1.0.0
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is not set. Usage: make release VERSION=v1.0.0"; exit 1; fi
	wails build -platform windows/amd64,windows/386 -nsis -ldflags "-X main.AppVersion=$(VERSION)"

# Default Go command
GO ?= go

# Build Lite version (Headless)
build-lite:
	@mkdir -p build/bin
	$(GO) build -ldflags "-X main.AppVersion=$(VERSION)" -o build/bin/ts-escpos-lite cmd/headless

# Build Lite version for Windows (amd64)
build-lite-windows:
	@mkdir -p build/bin
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "-X main.AppVersion=$(VERSION)" -o build/bin/ts-escpos-lite.exe ./cmd/headless

# Build Lite version for Windows (386)
build-lite-windows-386:
	@mkdir -p build/bin
	GOOS=windows GOARCH=386 $(GO) build -ldflags "-X main.AppVersion=$(VERSION)" -o build/bin/ts-escpos-lite-32.exe ./cmd/headless
