#!/bin/sh
set -eu

# monarchmoney-cli installer
# Usage: curl -fsSL https://raw.githubusercontent.com/thedavidweng/monarchmoney-cli/main/install.sh | sh

REPO="thedavidweng/monarchmoney-cli"
BINARY="monarch"
CASK="thedavidweng/tap/monarchmoney-cli"

step()  { printf '==> %s\n' "$1"; }
die()   { printf 'ERROR: %s\n' "$1" >&2; exit 1; }

# --- Detect OS and ARCH ---
os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) platform="darwin" ;;
  Linux)  platform="linux"  ;;
  *)      die "Unsupported OS: $os. Use install.ps1 on Windows." ;;
esac

case "$arch" in
  x86_64|amd64)  goarch="x86_64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *)             die "Unsupported architecture: $arch" ;;
esac

platform_label="$platform/$goarch"

# --- Resolve latest version ---
resolve_version() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/'
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O - "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/'
  else
    die "curl or wget is required."
  fi
}

download() {
  url="$1"
  output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$output" "$url"
  else
    die "curl or wget is required."
  fi
}

# --- Check for Homebrew ---
has_brew() {
  command -v brew >/dev/null 2>&1
}

install_via_brew() {
  step "Installing via Homebrew Cask (thedavidweng/tap)"
  brew tap "thedavidweng/tap" 2>/dev/null || true
  brew install --cask "$CASK"
}

install_binary() {
  version="$1"

  case "$platform" in
    darwin) asset="${BINARY}_darwin_universal.tar.gz" ;;
    linux)  asset="${BINARY}_linux_${goarch}.tar.gz" ;;
    *)      die "Unsupported platform: $platform" ;;
  esac

  url="https://github.com/$REPO/releases/download/$version/$asset"

  bin_dir="${MONARCH_INSTALL_DIR:-$HOME/.local/bin}"
  mkdir -p "$bin_dir"

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM

  step "Downloading $asset"
  download "$url" "$tmp_dir/$asset"

  step "Installing to $bin_dir/$BINARY"
  tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
  chmod +x "$tmp_dir/$BINARY"
  mv -f "$tmp_dir/$BINARY" "$bin_dir/$BINARY"

  # Add to PATH if needed
  case ":$PATH:" in
    *":$bin_dir:"*) ;;
    *)
      shell_profile=""
      case "$platform:${SHELL:-}" in
        darwin:*/zsh)  shell_profile="$HOME/.zprofile" ;;
        darwin:*/bash) shell_profile="$HOME/.bash_profile" ;;
        linux:*/zsh)   shell_profile="$HOME/.zshrc" ;;
        linux:*/bash)  shell_profile="$HOME/.bashrc" ;;
        *)             shell_profile="$HOME/.profile" ;;
      esac

      printf "\n# >>> monarchmoney-cli >>>\nexport PATH=\"%s:\$PATH\"\n# <<< monarchmoney-cli <<<\n" "$bin_dir" >> "$shell_profile"
      step "Added $bin_dir to PATH in $shell_profile"
      step "Run: export PATH=\"$bin_dir:\$PATH\" to use in current terminal"
      ;;
  esac

  step "Installed $("${bin_dir}/${BINARY}" --version 2>/dev/null || echo "$version")"
}

# --- Uninstall helper ---
uninstall_brew() {
  step "Uninstalling Homebrew-managed monarchmoney-cli"
  brew uninstall --cask "$CASK" 2>/dev/null || true
  brew untap "thedavidweng/tap" 2>/dev/null || true
}

uninstall_binary() {
  bin_dir="${MONARCH_INSTALL_DIR:-$HOME/.local/bin}"
  if [ -f "$bin_dir/$BINARY" ]; then
    step "Removing $bin_dir/$BINARY"
    rm -f "$bin_dir/$BINARY"
  fi
}

# --- Main ---
case "${1:-}" in
  uninstall)
    if has_brew && brew list --cask "monarchmoney-cli" >/dev/null 2>&1; then
      uninstall_brew
    elif has_brew && brew list --formula "monarchmoney-cli" >/dev/null 2>&1; then
      step "Uninstalling legacy Homebrew formula for monarchmoney-cli"
      brew uninstall --formula "monarchmoney-cli"
    else
      uninstall_binary
    fi
    step "Uninstalled. You may also remove monarchmoney-cli config from ~/.config/monarchmoney-cli/"
    exit 0
    ;;
  --help|-h)
    cat <<EOF
Usage: install.sh [uninstall]

Installs monarchmoney-cli. Prefers Homebrew Cask if available, otherwise
downloads the binary to ~/.local/bin.

Environment:
  MONARCH_INSTALL_DIR  Directory for binary (default: ~/.local/bin)

Options:
  uninstall    Remove monarchmoney-cli
  --help, -h   Show this help
EOF
    exit 0
    ;;
esac

step "Installing monarchmoney-cli ($platform_label)"

if has_brew; then
  install_via_brew
else
  version="$(resolve_version)"
  [ -z "$version" ] && die "Could not resolve latest version."
  step "Latest version: $version"
  install_binary "$version"
fi

printf '\n'
step "Run 'monarch auth login' to get started."
step "Run 'monarch --help' to see available commands."
