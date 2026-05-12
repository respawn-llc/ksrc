#!/usr/bin/env sh
set -eu

OUT="${KSRC_BUILD_OUT:-./bin/ksrc}"
VERSION="${KSRC_VERSION:-}"

usage() {
  cat <<EOF
Usage: build.sh [--out path]

Options:
  --out, --output, -o  Output binary path (default: ./bin/ksrc)

Environment:
  KSRC_BUILD_OUT  Output binary path
  KSRC_VERSION    Version override (default: VERSION file)
  CGO_ENABLED     CGO setting (default: 0)
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --out|--output|-o)
      OUT="${2:-}"
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

if [ -z "$OUT" ]; then
  echo "Output path cannot be empty." >&2
  exit 1
fi

if [ -z "$VERSION" ]; then
  VERSION="$(tr -d ' \n' < VERSION)"
fi

out_dir="$(dirname "$OUT")"
mkdir -p "$out_dir"

CGO_ENABLED="${CGO_ENABLED:-0}" go build \
  -trimpath \
  -buildvcs=false \
  -ldflags "-s -w -X github.com/respawn-app/ksrc/internal/cli.Version=${VERSION}" \
  -o "$OUT" \
  ./cmd/ksrc

echo "Built $OUT"
