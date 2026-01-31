#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/update-brew-tap.sh [--version vX.Y.Z] [--tap /path/to/homebrew-tap] [--repo owner/name] [--formula name] [--commit] [--push]

Updates the Homebrew tap formula for ksrc with a new tag tarball + sha256.

Defaults:
  --version : $KSRC_VERSION, $GITHUB_REF_NAME, or latest git tag in this repo
  --repo    : respawn-app/ksrc
  --formula : ksrc
  --tap     : $KSRC_TAP_PATH, $HOMEBREW_TAP_PATH, else ../homebrew-tap (relative to repo root)

Flags:
  --commit  : commit the formula update in the tap repo
  --push    : push the commit (implies --commit)
USAGE
}

version=""
repo="respawn-app/ksrc"
formula="ksrc"
tap_dir=""
do_commit="false"
do_push="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      version="$2"
      shift 2
      ;;
    --repo)
      repo="$2"
      shift 2
      ;;
    --formula)
      formula="$2"
      shift 2
      ;;
    --tap)
      tap_dir="$2"
      shift 2
      ;;
    --commit)
      do_commit="true"
      shift
      ;;
    --push)
      do_commit="true"
      do_push="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"
if [[ -z "$repo_root" ]]; then
  echo "Not inside a git repo" >&2
  exit 1
fi

if [[ -z "$version" ]]; then
  if [[ -n "${KSRC_VERSION:-}" ]]; then
    version="${KSRC_VERSION}"
  elif [[ -n "${GITHUB_REF_NAME:-}" ]]; then
    version="${GITHUB_REF_NAME}"
  elif [[ -n "${GITHUB_REF:-}" ]]; then
    version="${GITHUB_REF##*/}"
  else
    version="$(git -C "$repo_root" describe --tags --abbrev=0)"
  fi
fi

if [[ -z "$tap_dir" ]]; then
  if [[ -n "${KSRC_TAP_PATH:-}" ]]; then
    tap_dir="$KSRC_TAP_PATH"
  elif [[ -n "${HOMEBREW_TAP_PATH:-}" ]]; then
    tap_dir="$HOMEBREW_TAP_PATH"
  elif [[ -d "$repo_root/../homebrew-tap" ]]; then
    tap_dir="$repo_root/../homebrew-tap"
  else
    echo "Tap repo not found. Provide --tap or set HOMEBREW_TAP_PATH" >&2
    exit 1
  fi
fi

formula_path="$tap_dir/Formula/${formula}.rb"
if [[ ! -f "$formula_path" ]]; then
  echo "Formula not found: $formula_path" >&2
  exit 1
fi

url="https://github.com/${repo}/archive/refs/tags/${version}.tar.gz"

tmp_file="$(mktemp)"
cleanup() {
  if command -v trash >/dev/null 2>&1; then
    trash "$tmp_file" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

curl -fsSL "$url" -o "$tmp_file"
if command -v sha256sum >/dev/null 2>&1; then
  sha256="$(sha256sum "$tmp_file" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  sha256="$(shasum -a 256 "$tmp_file" | awk '{print $1}')"
else
  echo "sha256sum or shasum required" >&2
  exit 1
fi

perl -0pi -e "s|^  url \".*\"|  url \"$url\"|m" "$formula_path"
perl -0pi -e "s|^  sha256 \".*\"|  sha256 \"$sha256\"|m" "$formula_path"
# Drop stale bottle blocks; brew pr-pull will regenerate correct bottles.
perl -0pi -e 's/^\s*bottle do\n(?:.*\n)*?\s*end\n\n//m' "$formula_path"

if [[ "$do_commit" == "true" ]]; then
  git -C "$tap_dir" add "$formula_path"
  if git -C "$tap_dir" diff --cached --quiet; then
    echo "No formula changes to commit"
  else
    git -C "$tap_dir" commit -m "${formula} ${version}"
  fi
fi

if [[ "$do_push" == "true" ]]; then
  git -C "$tap_dir" push
fi

echo "Updated ${formula_path}"
echo "  url: $url"
echo "  sha256: $sha256"
