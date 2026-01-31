# Release workflow

This document describes how a release is cut, what must be bumped, and what automation runs.

## Files and locations

- Version source: `VERSION`
- Claude plugin version: `.claude-plugin/plugin.json`
- Release workflow: `.github/workflows/release.yml`
- Tap update script: `scripts/update-brew-tap.sh`
- Homebrew tap repo (separate): `../homebrew-tap` or https://github.com/respawn-app/homebrew-tap.
  - Formula: `../homebrew-tap/Formula/ksrc.rb`
  - Tap CI:
    - `../homebrew-tap/.github/workflows/tests.yml` (brew test-bot)
    - `../homebrew-tap/.github/workflows/publish.yml` (brew pr-pull)

## What to bump before a release

1. `VERSION`
   - This becomes the tag `vX.Y.Z` and the CLI version.
2. `.claude-plugin/plugin.json`
   - Keep plugin version aligned with the release version.

Optional (as needed):
- Update docs in `docs/` and skills in `skills/` when CLI flags, outputs, APIs or formats change.

## How a release is cut

Trigger the workflow:
- `gh workflow run release.yml --ref main`

What it does:
1. Reads `VERSION` and computes the tag `vX.Y.Z`.
2. Creates the git tag (if missing) and pushes it.
3. Builds release binaries for all OS/arch pairs and uploads them to a draft GitHub release.
4. Updates the Homebrew tap by opening a PR in `respawn-app/homebrew-tap`:
   - `scripts/update-brew-tap.sh` updates the source tarball URL + sha256.
   - The script also removes any existing `bottle do` block, so bottles are regenerated.

## Tap publishing (bottles)

Bottles are generated and uploaded by the tap repo:

1. The release workflow opens a tap PR with label `pr-pull`.
2. `../homebrew-tap/.github/workflows/publish.yml` runs `brew pr-pull` on that PR.
3. `brew pr-pull` builds bottles and writes the `bottle do` block back into the formula.
4. The PR is merged, and users can install via bottles by default.

Notes:
- `brew install -s ksrc` still builds from source because the formula points at the source tarball.
- If bottles become mismatched with the formula (version/revision/license), re-run pr-pull on a fresh PR with no bottle block.

## Post-release checks

- Verify the draft release assets are present and correct.
- Verify the tap PR is opened and `pr-pull` ran successfully.
- On macOS 26 Tahoe (arm64), run:
  - `brew tap respawn-app/tap`
  - `brew install ksrc`
