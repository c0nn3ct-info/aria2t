#!/bin/sh
# aria2t installer (macOS / Linux): downloads the latest release binary,
# verifies its checksum, and installs it to /usr/local/bin or ~/.local/bin.
set -eu

REPO="c0nn3ct-info/aria2t"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin | linux) ;;
  *)
    echo "unsupported OS: $os (use the Windows script or build from source)" >&2
    exit 1
    ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
  sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
[ -n "$tag" ] || { echo "could not resolve the latest release" >&2; exit 1; }

pkg="aria2t-$tag-$os-$arch.tar.gz"
url="https://github.com/$REPO/releases/download/$tag/$pkg"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading $pkg"
curl -fSL --progress-bar "$url" -o "$tmp/$pkg"
curl -fsSL "https://github.com/$REPO/releases/download/$tag/SHA256SUMS" -o "$tmp/SHA256SUMS"
(
  cd "$tmp"
  grep " $pkg\$" SHA256SUMS > sum.txt
  sha256sum -c sum.txt 2>/dev/null || shasum -a 256 -c sum.txt
) >/dev/null
echo "checksum OK"

tar -xzf "$tmp/$pkg" -C "$tmp"

dest=/usr/local/bin
[ -w "$dest" ] || dest="$HOME/.local/bin"
mkdir -p "$dest"
install -m 0755 "$tmp/aria2t-$tag-$os-$arch/aria2t" "$dest/aria2t"
echo "installed $dest/aria2t ($tag)"

case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "note: $dest is not on your PATH" ;;
esac
command -v aria2c >/dev/null 2>&1 ||
  echo "note: aria2c not found — the built-in daemon needs aria2 (brew install aria2 / sudo apt install aria2)"
