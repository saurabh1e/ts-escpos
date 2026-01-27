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

echo "🚀 Starting release process for version $VERSION..."

# 1. build the windows installer locally
echo "🔨 Building Windows Installer..."
make release VERSION=$VERSION

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# 2. Add binaries to git
echo "📦 Committing artifacts..."
git add -f build/bin/ts-escpos-amd64-installer.exe
git add -f build/bin/ts-escpos.exe

git commit -m "chore: release artifacts for $VERSION"

# 3. Create and push tag
echo "🏷️ Tagging version $VERSION..."
git tag $VERSION
git push origin $VERSION
git push origin HEAD

echo "✅ Done! GitHub Action should now trigger and create a release with the uploaded artifacts."
