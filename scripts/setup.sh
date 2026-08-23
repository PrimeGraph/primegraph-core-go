#!/bin/sh
# One-time setup for a fresh clone: point git at the repository's own hooks so
# the Conventional Commits check in .githooks/commit-msg runs on every commit.
#
# Usage: sh scripts/setup.sh

set -eu

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$REPO_ROOT"

git config core.hooksPath .githooks
chmod +x .githooks/* scripts/*.sh

echo "core.hooksPath is now $(git config core.hooksPath)"
