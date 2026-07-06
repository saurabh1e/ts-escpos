# Makefile for ts-escpos Wails project

.PHONY: all dev build build-windows build-windows-386 installer-windows deps clean release \
	build-normal-64 build-normal-32 build-lite-64 build-lite-32 build-all \
	build-lite build-lite-windows build-lite-windows-386 \
	installer-normal-64 installer-normal-32 installer-lite-64 installer-lite-32 installer-all \
	sign-bins sign-installers sign-all

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

# Version flags. When VERSION is unset, pass no override so the build falls back
# to the default AppVersion baked into the source (and the .nsi files use their
# own defaults — an empty -DINFO_PRODUCTVERSION= breaks VIFileVersion).
CLEAN_VERSION := $(VERSION:v%=%)
NSIS_VERSION_DEF := $(if $(CLEAN_VERSION),-DINFO_PRODUCTVERSION=$(CLEAN_VERSION),)
LDFLAGS := $(if $(VERSION),-ldflags "-X main.AppVersion=$(VERSION)",)

# Lite (headless) binaries are built with plain `go build`, which does NOT strip
# by default. -s (symbol table) and -w (DWARF) shrink the binary ~25-30%;
# -trimpath removes local filesystem paths. (Wails GUI builds already strip in
# production, so this only applies to the lite targets.)
LITE_LDFLAGS := -s -w $(if $(VERSION),-X main.AppVersion=$(VERSION),)

# Build Lite version (Headless)
build-lite:
	@mkdir -p build/bin
	$(GO) build -trimpath -ldflags "$(LITE_LDFLAGS)" -o build/bin/ts-escpos-lite cmd/headless

# Build Lite version for Windows (amd64)
build-lite-windows:
	@mkdir -p build/bin
	GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LITE_LDFLAGS)" -o build/bin/ts-escpos-lite.exe ./cmd/headless

# Build Lite version for Windows (386)
build-lite-windows-386:
	@mkdir -p build/bin
	GOOS=windows GOARCH=386 $(GO) build -trimpath -ldflags "$(LITE_LDFLAGS)" -o build/bin/ts-escpos-lite-32.exe ./cmd/headless

# --- Convenience build options: normal & lite, 64-bit & 32-bit (Windows) ---

# Normal (Wails GUI) builds
build-normal-64: build-windows
build-normal-32: build-windows-386

# Lite (headless) builds
build-lite-64: build-lite-windows
build-lite-32: build-lite-windows-386

# Build every variant: normal + lite, 64-bit + 32-bit
build-all: build-normal-64 build-normal-32 build-lite-64 build-lite-32

# --- Installers (Windows, NSIS via makensis) ---
# Requires NSIS installed (macOS: brew install nsis).
# Pass VERSION=vX.Y.Z to stamp installer/binary metadata.

# Normal (GUI) installer, 64-bit -> build/bin/ts-escpos-amd64-installer.exe
installer-normal-64:
	wails build -platform windows/amd64 -trimpath -nsis $(LDFLAGS)

# Normal (GUI) installer, 32-bit -> build/bin/ts-escpos-386-installer.exe
installer-normal-32:
	wails build -platform windows/386 -trimpath -o ts-escpos-386.exe $(LDFLAGS)
	cd build/windows/installer && makensis $(NSIS_VERSION_DEF) project_386.nsi

# Lite (headless) installer, 64-bit -> build/bin/ts-escpos-lite-amd64-setup.exe
installer-lite-64: build-lite-windows
	cd build/windows/installer && makensis $(NSIS_VERSION_DEF) -DLITE_BINARY="..\..\bin\ts-escpos-lite.exe" -DLITE_ARCH="amd64" project_lite.nsi

# Lite (headless) installer, 32-bit -> build/bin/ts-escpos-lite-386-setup.exe
installer-lite-32: build-lite-windows-386
	cd build/windows/installer && makensis $(NSIS_VERSION_DEF) -DLITE_BINARY="..\..\bin\ts-escpos-lite-32.exe" -DLITE_ARCH="386" project_lite.nsi

# Build every installer: normal + lite, 64-bit + 32-bit
installer-all: installer-normal-64 installer-normal-32 installer-lite-64 installer-lite-32

# --- Code signing (Authenticode) ---
# Unsigned Go/Wails binaries are routinely flagged by Windows Defender/SmartScreen
# regardless of build flags. Authenticode-signing with a real certificate is the
# only reliable fix; strip + no-UPX (already applied) only reduce heuristic hits.
#
# Signs Windows PE files from macOS/Linux via osslsigncode:
#     brew install osslsigncode
# Provide a cert (targets are a SAFE NO-OP if SIGN_CERT is unset):
#     make sign-all SIGN_CERT=cert.pfx SIGN_PASS=secret
#
# IMPORTANT ordering for installers: sign the app binaries FIRST, then build the
# installers (so the packaged exe is signed), then sign the installers:
#     make build-normal-64 build-normal-32 build-lite-windows build-lite-windows-386
#     make sign-bins SIGN_CERT=... SIGN_PASS=...
#     make installer-all
#     make sign-installers SIGN_CERT=... SIGN_PASS=...
SIGN_CERT ?=
SIGN_PASS ?=
SIGN_TS   ?= http://timestamp.digicert.com
SIGN_NAME ?= ts-escpos
SIGN_URL  ?= https://github.com/saurabh1e/ts-escpos

# $(call do_sign,<glob>) — sign matching files in place, or skip if no cert.
define do_sign
	@if [ -z "$(SIGN_CERT)" ]; then \
		echo "⚠️  SIGN_CERT not set — skipping signing ($(1)). Once you have a cert:"; \
		echo "    make $@ SIGN_CERT=cert.pfx SIGN_PASS=secret"; \
	else \
		command -v osslsigncode >/dev/null 2>&1 || { echo "❌ osslsigncode not installed (brew install osslsigncode)"; exit 1; }; \
		for f in $(1); do \
			[ -f "$$f" ] || continue; \
			echo "🔏 Signing $$f"; \
			osslsigncode sign -pkcs12 "$(SIGN_CERT)" -pass "$(SIGN_PASS)" \
				-n "$(SIGN_NAME)" -i "$(SIGN_URL)" -ts "$(SIGN_TS)" -h sha256 \
				-in "$$f" -out "$$f.signed" && mv "$$f.signed" "$$f" || exit 1; \
		done; \
	fi
endef

# Sign the application/headless binaries (run BEFORE building installers).
sign-bins:
	$(call do_sign,build/bin/ts-escpos-amd64.exe build/bin/ts-escpos-386.exe build/bin/ts-escpos.exe build/bin/ts-escpos-lite.exe build/bin/ts-escpos-lite-32.exe)

# Sign the installer executables (run AFTER building installers).
sign-installers:
	$(call do_sign,build/bin/ts-escpos-amd64-installer.exe build/bin/ts-escpos-386-installer.exe build/bin/ts-escpos-lite-amd64-setup.exe build/bin/ts-escpos-lite-386-setup.exe)

# Sign everything currently in build/bin (convenience; prefer the ordered flow above).
sign-all:
	$(call do_sign,build/bin/*.exe)
