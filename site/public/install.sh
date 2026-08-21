#!/bin/sh
# aria2t installer (macOS / Linux): downloads the latest release binary,
# verifies its checksum, and installs it to /usr/local/bin or ~/.local/bin.
#
# Pass a Chrome extension id to ALSO register the browser extension's native
# messaging host — the same aria2t binary serves both the TUI and the
# extension, so there is no separate helper to install:
#   curl -fsSL https://aria2t.c0nn3ct.info/install.sh | sh -s -- <extension-id>
set -eu

REPO="c0nn3ct-info/aria2t"
EXT_ID="${1:-}"

# Validate the extension id up front (a 32-char a-p string) so a typo fails
# before anything is downloaded.
if [ -n "$EXT_ID" ]; then
  case "$EXT_ID" in
    *[!a-p]*) echo "invalid extension id: $EXT_ID (expected 32 chars a-p)" >&2; exit 1 ;;
  esac
  [ "${#EXT_ID}" -eq 32 ] ||
    { echo "invalid extension id: $EXT_ID (expected 32 chars a-p)" >&2; exit 1; }
fi

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

# ---------------------------------------------------------------------------
# Optional: register the browser extension's native messaging host. Points the
# manifest at the aria2t binary just installed — it runs as the host when
# Chrome launches it with the extension origin.
[ -n "$EXT_ID" ] || exit 0

NM_NAME="com.aria2t.host"
HOST_BIN="$dest/aria2t"

# Merge ids into allowed_origins instead of overwriting: each browser/profile
# has its own extension id, so re-running from a second browser must not evict
# the first. Union of (ids already in the file) + the passed id, deduped.
build_origins() {                       # $1 = manifest path, $2 = ext id
  { [ -f "$1" ] && grep -oE 'chrome-extension://[a-p]{32}/' "$1"
    echo "chrome-extension://$2/"; } | sort -u |
    awk 'NR>1{printf ",\n    "} {printf "\"%s\"", $0}'
}

if [ "$os" = darwin ]; then
  dirs=$(cat <<EOF
$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts
$HOME/Library/Application Support/Google/Chrome Beta/NativeMessagingHosts
$HOME/Library/Application Support/Google/Chrome Canary/NativeMessagingHosts
$HOME/Library/Application Support/Chromium/NativeMessagingHosts
$HOME/Library/Application Support/BraveSoftware/Brave-Browser/NativeMessagingHosts
$HOME/Library/Application Support/Microsoft Edge/NativeMessagingHosts
$HOME/Library/Application Support/Arc/User Data/NativeMessagingHosts
$HOME/Library/Application Support/Vivaldi/NativeMessagingHosts
$HOME/Library/Application Support/com.operasoftware.Opera/NativeMessagingHosts
$HOME/Library/Application Support/Yandex/YandexBrowser/NativeMessagingHosts
EOF
)
else
  base="${XDG_CONFIG_HOME:-$HOME/.config}"
  dirs=$(cat <<EOF
$base/google-chrome/NativeMessagingHosts
$base/google-chrome-beta/NativeMessagingHosts
$base/google-chrome-unstable/NativeMessagingHosts
$base/chromium/NativeMessagingHosts
$base/BraveSoftware/Brave-Browser/NativeMessagingHosts
$base/microsoft-edge/NativeMessagingHosts
$base/vivaldi/NativeMessagingHosts
$base/opera/NativeMessagingHosts
$base/yandex-browser/NativeMessagingHosts
EOF
)
fi

# Heredoc feeds the loop directly (no pipeline) so `written` survives; the
# per-browser dir may contain spaces, so read whole lines.
written=0
while IFS= read -r dir; do
  [ -n "$dir" ] || continue
  parent=$(dirname "$dir")
  [ -d "$parent" ] || continue          # only browsers that are actually installed
  mkdir -p "$dir"
  manifest="$dir/$NM_NAME.json"
  origins=$(build_origins "$manifest" "$EXT_ID")
  cat > "$manifest" <<JSON
{
  "name": "$NM_NAME",
  "description": "aria2t native host",
  "path": "$HOST_BIN",
  "type": "stdio",
  "allowed_origins": [
    $origins
  ]
}
JSON
  echo "  registered $manifest"
  written=$((written + 1))
done <<EOF
$dirs
EOF

if [ "$written" -eq 0 ]; then
  echo "no supported browser data dirs found — open your browser once, then re-run" >&2
  exit 1
fi
echo "extension host registered for $written browser(s); reload aria2t on chrome://extensions"
