#!/usr/bin/env sh
set -eu

usage() {
  cat <<EOF
Usage: update-dependencies.sh [--verify]

Updates committed dependency version sources:
  - Go toolchain and modules in go.mod/go.sum
  - Gradle wrappers in sample/ and testdata/integration/
  - Gradle version catalog in sample/gradle/libs.versions.toml
  - Gradle fixture dependency pins
  - Pinned GitHub Actions in .github/workflows/*.yml

Options:
  --verify      Run format/vet/test/build plus sample smoke check after updating

Environment:
  GITHUB_TOKEN  Optional token for GitHub tag API rate limits
EOF
}

verify=false

while [ $# -gt 0 ]; do
  case "$1" in
    --verify)
      verify=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$repo_root"

echo "Updating Go toolchain and modules..."
go get go@latest
go get -u ./...
go mod tidy

echo "Updating Gradle wrappers..."
./sample/gradlew -p sample wrapper --gradle-version latest --distribution-type bin
./testdata/integration/gradlew -p testdata/integration wrapper --gradle-version latest --distribution-type bin

echo "Updating sample Gradle version catalog..."
./sample/gradlew -p sample -Dorg.gradle.unsafe.isolated-projects=false --no-configuration-cache versionCatalogUpdate
go run ./scripts/sync-gradle-fixture-deps.go sample/gradle/libs.versions.toml testdata/integration/build.gradle

echo "Updating GitHub Actions..."
go run ./scripts/update-github-actions.go .github/workflows

if [ "$verify" = true ]; then
  echo "Verifying dependency update..."
  ./scripts/ci-format.sh
  go vet ./...
  go test ./...
  ./scripts/build.sh
  ./bin/ksrc search 'public actual class LocalDate' --module org.jetbrains.kotlinx:kotlinx-datetime --project ./sample >/dev/null
fi

echo "Dependency update complete."
