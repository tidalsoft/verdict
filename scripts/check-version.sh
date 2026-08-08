#!/bin/sh
# check-version.sh — verify that the Version const in version.go matches the
# git tag that triggered CI.
#
# This is intentionally a shell script, not a Go test or binary.  The repo's
# 100%-coverage gate would require every new package to be fully covered, which
# forces us to either write a coverage-excluded test harness or to give up on
# testing the check itself.  A POSIX shell script side-steps that entirely: CI
# invokes it directly, and the exit code is the assertion.
#
# Usage: check-version.sh <tag-ref>   (e.g. "v0.1.0")

set -e

# ---------------------------------------------------------------------------
# Argument check
# ---------------------------------------------------------------------------
if [ $# -lt 1 ]; then
  echo "usage: check-version.sh <tag-ref>" >&2
  echo "  e.g. check-version.sh v0.1.0" >&2
  exit 1
fi

TAG="$1"

# ---------------------------------------------------------------------------
# Locate version.go (relative to repo root, which is the script's cwd)
# ---------------------------------------------------------------------------
VERSION_FILE="version.go"

if [ ! -f "$VERSION_FILE" ]; then
  echo "error: $VERSION_FILE not found — expected at repo root" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Extract the Version const value.
# Match: const Version = "X.Y.Z"
# ---------------------------------------------------------------------------
CONST=$(sed -n 's/^const Version = "\(.*\)"/\1/p' "$VERSION_FILE")

if [ -z "$CONST" ]; then
  echo "error: could not extract Version const from $VERSION_FILE" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Compare "v${CONST}" to the tag argument.
# Tags use a leading v (v0.1.0); the const does not (0.1.0).
# ---------------------------------------------------------------------------
EXPECTED="v${CONST}"

if [ "$EXPECTED" = "$TAG" ]; then
  echo "ok: version const ($CONST) matches tag ($TAG)"
  exit 0
else
  echo "mismatch: version const is \"$CONST\" (expected tag \"$EXPECTED\"), got tag \"$TAG\"" >&2
  exit 1
fi
