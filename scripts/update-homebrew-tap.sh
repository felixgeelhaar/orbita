#!/bin/bash
# Update Homebrew tap with new release
set -euo pipefail

VERSION="${1:-}"
HOMEBREW_TAP_TOKEN="${HOMEBREW_TAP_TOKEN:-}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.3.0"
    exit 1
fi

if [ -z "$HOMEBREW_TAP_TOKEN" ]; then
    echo "Error: HOMEBREW_TAP_TOKEN environment variable not set"
    exit 1
fi

# Calculate SHA256 hashes for each platform
echo "Calculating SHA256 hashes..."
SHA_DARWIN_ARM64=$(sha256sum dist/orbita-darwin-arm64.tar.gz | awk '{print $1}')
SHA_DARWIN_AMD64=$(sha256sum dist/orbita-darwin-amd64.tar.gz | awk '{print $1}')
SHA_LINUX_ARM64=$(sha256sum dist/orbita-linux-arm64.tar.gz | awk '{print $1}')
SHA_LINUX_AMD64=$(sha256sum dist/orbita-linux-amd64.tar.gz | awk '{print $1}')

echo "SHA256 hashes:"
echo "  darwin-arm64: $SHA_DARWIN_ARM64"
echo "  darwin-amd64: $SHA_DARWIN_AMD64"
echo "  linux-arm64:  $SHA_LINUX_ARM64"
echo "  linux-amd64:  $SHA_LINUX_AMD64"

# Clone homebrew-tap
echo "Cloning homebrew-tap..."
rm -rf tap
git clone "https://x-access-token:${HOMEBREW_TAP_TOKEN}@github.com/felixgeelhaar/homebrew-tap.git" tap
cd tap

# Generate updated formula
echo "Generating formula..."
cat > Formula/orbita.rb << EOF
# Homebrew formula for Orbita
# To install: brew tap felixgeelhaar/tap && brew install orbita
class Orbita < Formula
  desc "CLI-first adaptive productivity operating system - orchestrates tasks, calendars, habits, and meetings"
  homepage "https://github.com/felixgeelhaar/orbita"
  version "${VERSION}"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/felixgeelhaar/orbita/releases/download/v#{version}/orbita-darwin-arm64.tar.gz"
      sha256 "${SHA_DARWIN_ARM64}"

      def install
        bin.install "orbita-darwin-arm64" => "orbita"
      end
    else
      url "https://github.com/felixgeelhaar/orbita/releases/download/v#{version}/orbita-darwin-amd64.tar.gz"
      sha256 "${SHA_DARWIN_AMD64}"

      def install
        bin.install "orbita-darwin-amd64" => "orbita"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/felixgeelhaar/orbita/releases/download/v#{version}/orbita-linux-arm64.tar.gz"
      sha256 "${SHA_LINUX_ARM64}"

      def install
        bin.install "orbita-linux-arm64" => "orbita"
      end
    else
      url "https://github.com/felixgeelhaar/orbita/releases/download/v#{version}/orbita-linux-amd64.tar.gz"
      sha256 "${SHA_LINUX_AMD64}"

      def install
        bin.install "orbita-linux-amd64" => "orbita"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/orbita version")
  end
end
EOF

# Commit and push
echo "Committing changes..."
git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"
git add Formula/orbita.rb
git commit -m "orbita: update to v${VERSION}"
git push

echo "Done! Homebrew tap updated to v${VERSION}"
