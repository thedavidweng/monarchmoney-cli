class MonarchmoneyCli < Formula
  desc "A local, agent-friendly CLI for Monarch Money."
  homepage "https://github.com/monarchmoney-cli/monarch"
  url "https://github.com/monarchmoney-cli/monarch/releases/download/v0.1.0/monarch_Darwin_arm64.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256"
  license "MIT"

  def install
    bin.install "monarch"
  end

  test do
    system "#{bin}/monarch", "--version"
  end
end
