#!/bin/bash
set -e

echo "🔄 Syncing SOURCE CODE ONLY from GitLab to GitHub..."
echo "📦 Binaries are excluded - they stay as GitLab artifacts"

# Check if we're on main branch
if [[ $(git branch --show-current) != "main" ]]; then
    echo "❌ Error: Must be on main branch to sync"
    exit 1
fi

# Ensure no binaries are staged or committed
echo "🧹 Checking for any binary files..."
if git ls-files | grep -E '\.(exe|dll|so|dylib|zip|tar\.gz)$'; then
    echo "❌ Error: Binary files detected! Please remove them before syncing."
    exit 1
fi

# Fetch latest from GitLab origin
echo "📥 Fetching latest from GitLab..."
git fetch origin

# Push to GitHub (force to ensure sync) - SOURCE CODE ONLY
echo "📤 Pushing SOURCE CODE to GitHub..."
git push github main --force
git push github --tags --force

echo "✅ Sync complete! GitLab → GitHub (SOURCE CODE ONLY)"
echo "🌐 GitHub repo: https://github.com/nrp-nautilus/terraform-provider-kubevirt"
echo "📦 Binaries remain as GitLab artifacts for download"
