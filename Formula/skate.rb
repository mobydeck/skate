# This file is a template. Generated formula is in dist/skate.rb after `just brew-formula`.
class Skate < Formula
  desc "Access Mattermost Boards tasks from CLI and AI agents via MCP"
  homepage "https://github.com/mobydeck/skate"
  version "VERSION"
  license "MIT"

  on_macos do
    url "https://github.com/mobydeck/skate/releases/download/VERSION/skate-darwin-arm64"
    sha256 "SHA256_DARWIN_ARM64"
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/mobydeck/skate/releases/download/VERSION/skate-linux-arm64"
      sha256 "SHA256_LINUX_ARM64"
    else
      url "https://github.com/mobydeck/skate/releases/download/VERSION/skate-linux-amd64"
      sha256 "SHA256_LINUX_AMD64"
    end
  end

  def install
    binary = Dir["skate-*"].first || "skate"
    bin.install binary => "skate"
  end

  test do
    assert_match "skate version", shell_output("#{bin}/skate version")
  end
end
