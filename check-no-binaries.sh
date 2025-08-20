#!/bin/bash
set -e

echo "🔍 Checking for binary files in repository..."

# Check for common binary extensions
BINARY_FILES=$(git ls-files | grep -E '\.(exe|dll|so|dylib|zip|tar\.gz|tar|bin|obj|o|a)$' || true)

if [[ -n "$BINARY_FILES" ]]; then
    echo "❌ Binary files detected:"
    echo "$BINARY_FILES"
    echo ""
    echo "🚫 These files should NOT be committed to the repository!"
    echo "💡 They should be:"
    echo "   - Added to .gitignore"
    echo "   - Stored as GitLab CI/CD artifacts"
    echo "   - Downloaded from GitLab releases"
    exit 1
else
    echo "✅ No binary files detected - repository is clean!"
    echo "🚀 Safe to sync to GitHub"
fi
