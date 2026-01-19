#!/usr/bin/env sh
set -eu

REPO="${KSRC_REPO:-respawn-app/ksrc}"
VERSION="${KSRC_VERSION:-${VERSION:-}}"
PREFIX="${KSRC_PREFIX:-}"
RELEASE_BASE="${KSRC_RELEASE_BASE:-https://github.com/${REPO}/releases/download}"

usage() {
  cat <<EOF
Usage: install.sh [--version vX.Y.Z|X.Y.Z] [--prefix /path]

Options:
  --version  Release tag to install (vX.Y.Z or X.Y.Z; defaults to latest)
  --prefix   Install prefix (defaults to /usr/local or ~/.local)

Environment:
  KSRC_VERSION       Override version
  KSRC_PREFIX        Override prefix
  KSRC_REPO          Override repo (default: respawn-app/ksrc)
  KSRC_RELEASE_BASE  Override release base URL
  GITHUB_TOKEN       GitHub token for API rate limits
  GH_TOKEN           GitHub token for API rate limits
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --prefix)
      PREFIX="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

if [ -z "$VERSION" ]; then
  api_url="https://api.github.com/repos/${REPO}/releases/latest"
  auth_header=""
  api_headers="$tmpdir/api.headers"
  api_body="$tmpdir/api.json"
  api_status=""
  api_message=""
  api_rate_limited="0"
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    auth_header="Authorization: Bearer ${GITHUB_TOKEN}"
  elif [ -n "${GH_TOKEN:-}" ]; then
    auth_header="Authorization: Bearer ${GH_TOKEN}"
  fi
  if [ -n "$auth_header" ]; then
    api_status="$(curl -sSL -D "$api_headers" -o "$api_body" -w '%{http_code}' -H "$auth_header" "$api_url" || true)"
  else
    api_status="$(curl -sSL -D "$api_headers" -o "$api_body" -w '%{http_code}' "$api_url" || true)"
  fi
  if [ "$api_status" = "200" ]; then
    VERSION="$(awk -F'"' '/"tag_name":/ {print $4; exit}' "$api_body")"
  else
    api_message="$(awk -F'"' '/"message":/ {print $4; exit}' "$api_body" || true)"
    if [ -n "$api_message" ] && echo "$api_message" | grep -qi "API rate limit exceeded"; then
      api_rate_limited="1"
    elif grep -qi '^x-ratelimit-remaining: 0' "$api_headers" 2>/dev/null; then
      api_rate_limited="1"
    fi
  fi
fi
if [ -z "$VERSION" ]; then
  latest_url="$(curl -sSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" || true)"
  case "$latest_url" in
    */releases/tag/*) VERSION="${latest_url##*/tag/}" ;;
  esac
fi
if [ -z "$VERSION" ]; then
  if [ "${api_rate_limited:-0}" = "1" ]; then
    echo "GitHub API rate limit exceeded. Set GITHUB_TOKEN or GH_TOKEN and retry." >&2
  elif [ -n "${api_message:-}" ]; then
    echo "$api_message" >&2
  fi
  echo "Failed to resolve latest version." >&2
  exit 1
fi

tag="$VERSION"
if [ "${VERSION#v}" = "$VERSION" ]; then
  tag="v$VERSION"
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin) os="darwin" ;;
  linux) os="linux" ;;
  mingw*|msys*|cygwin*) os="windows" ;;
  *)
    echo "Unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

ver="${tag#v}"
base_name="ksrc_${ver}_${os}_${arch}"
if [ "$os" = "windows" ]; then
  archive="${base_name}.zip"
  bin_name="${base_name}.exe"
else
  archive="${base_name}.tar.gz"
  bin_name="${base_name}"
fi

url="${RELEASE_BASE}/${tag}/${archive}"
download_status="$(curl -sSL -o "$tmpdir/$archive" -w '%{http_code}' "$url" || true)"
if [ "$download_status" != "200" ]; then
  if [ "$download_status" = "404" ]; then
    echo "Version not found. Try without --version or use vX.Y.Z / X.Y.Z." >&2
  fi
  echo "Failed to download ${archive} (HTTP ${download_status})." >&2
  exit 1
fi

if [ "$os" = "windows" ]; then
  if ! command -v unzip >/dev/null 2>&1; then
    echo "unzip is required to install on Windows." >&2
    exit 1
  fi
  unzip -q "$tmpdir/$archive" -d "$tmpdir"
else
  tar -xzf "$tmpdir/$archive" -C "$tmpdir"
fi

if [ -z "$PREFIX" ]; then
  if [ -w /usr/local/bin ]; then
    PREFIX="/usr/local"
  else
    PREFIX="$HOME/.local"
  fi
fi

bin_dir="$PREFIX"
case "$bin_dir" in
  */bin) ;;
  *) bin_dir="${bin_dir%/}/bin" ;;
esac

mkdir -p "$bin_dir"
target="$bin_dir/ksrc"
if [ -e "$target" ]; then
  if [ -d "$target" ]; then
    echo "Refusing to overwrite directory $target" >&2
    exit 1
  fi
  if [ -L "$target" ]; then
    echo "Refusing to overwrite symlink $target" >&2
    exit 1
  fi
  echo "Warning: overwriting existing $target" >&2
fi
install -m 755 "$tmpdir/$bin_name" "$target"

echo "Installed ksrc to $target"
if ! echo "$PATH" | tr ':' '\n' | grep -q "^${bin_dir}$"; then
  echo "Note: $bin_dir is not on PATH. Add it to your shell profile."
fi
