#!/usr/bin/env bash
# The Go module's coverage gate: every function at 100.0% of statements.
#
# Go has no threshold setting the way vitest does, so this script is the config:
# it is what `.github/workflows/ci.yml` runs, and what the `verify` skill points
# at. The bar is per **function**, not the total — a 99.9% total is one function
# nobody tested hiding behind several hundred that are, which is exactly the
# hole a whole-module average is built to conceal.
#
#   tui/scripts/coverage.sh              # writes cover.out
#   tui/scripts/coverage.sh /tmp/x.out   # somewhere else
#
# `-count=1` is not optional: a cached test result contributes nothing to the
# merged profile, so a warm cache reports functions as uncovered that are not.
# `-coverpkg=./...` is what makes a package's code count when another package's
# test is what exercises it.
set -euo pipefail

cd "$(dirname "$0")/.."
profile="${1:-cover.out}"

go test ./... -count=1 -coverprofile="$profile" -coverpkg=./...

short="$(go tool cover -func="$profile" | awk '$3 != "100.0%"')"
if [ -n "$short" ]; then
  echo >&2
  echo "coverage: these are not at 100.0% —" >&2
  echo "$short" >&2
  echo >&2
  echo "Add the test in the same change. Code that turns out to be unreachable" >&2
  echo "gets restructured, not ignored — see CLAUDE.md for the three patterns." >&2
  exit 1
fi

echo "coverage: every function at 100.0%."
