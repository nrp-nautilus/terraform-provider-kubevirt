#!/bin/bash

# Script to release the Terraform provider with proper GPG signatures
set -e

# Configuration
PROVIDER_NAME="kubevirt"
NAMESPACE="terraform-dev"
VERSION=${1:-"v0.2.26"}

if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "❌ Invalid version format. Use: v0.2.26"
    exit 1
fi

echo "🚀 Releasing Terraform Provider $PROVIDER_NAME version $VERSION"

# Check if we're on main branch
if [[ $(git branch --show-current) != "main" ]]; then
    echo "❌ Must be on main branch to release"
    exit 1
fi

# Check if tag already exists
if git tag -l | grep -q "^$VERSION$"; then
    echo "❌ Tag $VERSION already exists"
    exit 1
fi

# Check if working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    echo "❌ Working directory is not clean. Commit or stash changes first."
    exit 1
fi

echo "✅ Pre-release checks passed"

# Build the provider
echo "🔨 Building provider..."
mkdir -p bin

# Build for all platforms
GOOS=linux GOARCH=amd64 go build -o bin/terraform-provider-kubevirt-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o bin/terraform-provider-kubevirt-linux-arm64 .
GOOS=darwin GOARCH=amd64 go build -o bin/terraform-provider-kubevirt-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o bin/terraform-provider-kubevirt-darwin-arm64 .

echo "✅ Build completed"

# Create zip files
echo "📦 Creating zip files..."
cd bin

# Create zip files with correct binary names
zip "terraform-provider-${PROVIDER_NAME}_${VERSION#v}_linux_amd64.zip" terraform-provider-kubevirt-linux-amd64
zip "terraform-provider-${PROVIDER_NAME}_${VERSION#v}_linux_arm64.zip" terraform-provider-kubevirt-linux-arm64
zip "terraform-provider-${PROVIDER_NAME}_${VERSION#v}_darwin_amd64.zip" terraform-provider-kubevirt-darwin-amd64
zip "terraform-provider-${PROVIDER_NAME}_${VERSION#v}_darwin_arm64.zip" terraform-provider-kubevirt-darwin-arm64

# Rename binaries inside zip files to correct Terraform provider name
echo "🔄 Renaming binaries inside zip files..."
for zipfile in *.zip; do
    echo "Processing $zipfile..."
    # Extract the zip
    unzip -o -q "$zipfile"
    # Get the binary name
    BINARY_NAME=$(find . -name "terraform-provider-kubevirt-*" -type f | head -1)
    # Rename it to the correct Terraform provider name
    mv "$BINARY_NAME" terraform-provider-kubevirt
    # Recreate the zip with the correct binary name
    rm "$zipfile"
    zip "$zipfile" terraform-provider-kubevirt
    # Clean up extracted files
    rm terraform-provider-kubevirt
done

# Copy manifest file
echo "📋 Copying manifest file..."
cp ../terraform-registry-manifest.json .

# Generate checksums
echo "🔐 Generating SHA256 checksums..."
shasum -a 256 *.zip > "terraform-provider-${PROVIDER_NAME}_${VERSION#v}_SHA256SUMS"

# Sign the checksums file
echo "✍️  Signing checksums file..."
gpg --batch --yes --local-user 1939A822A74AF28E --detach-sign "terraform-provider-${PROVIDER_NAME}_${VERSION#v}_SHA256SUMS"

echo "✅ Release artifacts created:"
ls -la

cd ..

# Update manifest with new version
echo "📝 Updating manifest with new version..."
python3 -c "
import json
import sys

# Read current manifest
with open('terraform-registry-manifest.json', 'r') as f:
    manifest = json.load(f)

# Find the provider
provider = None
for p in manifest['providers']:
    if p['name'] == '${PROVIDER_NAME}':
        provider = p
        break

if not provider:
    print('❌ Provider not found in manifest')
    sys.exit(1)

# Add new version
new_version = {
    'version': '${VERSION#v}',
    'protocols': ['5.0'],
    'platforms': [
        {'os': 'linux', 'arch': 'amd64'},
        {'os': 'linux', 'arch': 'arm64'},
        {'os': 'darwin', 'arch': 'amd64'},
        {'os': 'darwin', 'arch': 'arm64'}
    ]
}

# Check if version already exists
version_exists = False
for v in provider['versions']:
    if v['version'] == '${VERSION#v}':
        version_exists = True
        break

if not version_exists:
    provider['versions'].append(new_version)
    print('✅ Added version ${VERSION#v} to manifest')
else:
    print('⚠️  Version ${VERSION#v} already exists in manifest')

# Write updated manifest
with open('terraform-registry-manifest.json', 'w') as f:
    json.dump(manifest, f, indent=4)

print('✅ Manifest updated')
"

# Commit and tag
echo "🏷️  Creating git tag..."
git add terraform-registry-manifest.json
git commit -m "Release $VERSION - Update manifest" || echo "No changes to commit"

git tag -a "$VERSION" -m "Release $VERSION"

echo "✅ Release $VERSION prepared!"
echo ""
echo "📋 Next steps:"
echo "1. Review the changes: git diff HEAD~1"
echo "2. Push the tag: git push origin $VERSION"
echo "3. Push the commit: git push origin main"
echo "4. Monitor GitHub Actions for release creation"
echo "5. Verify HCP acceptance"
echo ""
echo "🎯 Files created in bin/ directory:"
ls -la bin/
