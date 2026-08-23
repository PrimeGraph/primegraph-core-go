#!/bin/sh
# Release github.com/primegraph/primegraph-core-go.
#
# Usage: sh scripts/release.sh <semver>       e.g. sh scripts/release.sh 1.4.0
#
# The argument is a bare semver with no leading "v"; the git tag gets the "v".
# Go records no version in a manifest and there is no registry to publish to —
# the module proxy serves whatever the tag points at — so this release is
# tag-only. Every check that can fail runs before the tag is created.
#
# The module path must never gain a /vN suffix, and this script must never add
# one. In Go, github.com/primegraph/primegraph-core-go/v2 is a *different*
# module path from github.com/primegraph/primegraph-core-go, and the toolchain
# will happily link both majors into one binary. That would give a graph two
# copies of every shared type again — the exact duplicate-nominal-type problem
# this module exists to remove. Stay on v0/v1 forever and make breaking changes
# by coordinating the generated packages instead.

set -eu

usage() {
  echo "Usage: sh scripts/release.sh <semver>   (bare semver, no leading 'v')" >&2
  exit 1
}

[ "$#" -eq 1 ] || usage
VERSION="$1"

SEMVER_RE='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
if ! printf '%s' "$VERSION" | grep -Eq "$SEMVER_RE"; then
  echo "release: '$VERSION' is not a bare semver (expected 1.4.0, not v1.4.0)" >&2
  exit 1
fi
TAG="v$VERSION"

MAJOR=${VERSION%%.*}
if [ "$MAJOR" -ge 2 ]; then
  echo "release: refusing major $MAJOR — it would require a /v$MAJOR module path," >&2
  echo "release: which Go treats as a separate module and links alongside this one." >&2
  exit 1
fi

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$REPO_ROOT"

# --- preflight --------------------------------------------------------------

DEFAULT_BRANCH=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's#^origin/##' || true)
[ -n "$DEFAULT_BRANCH" ] || DEFAULT_BRANCH=main

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "$DEFAULT_BRANCH" ]; then
  echo "release: on branch '$CURRENT_BRANCH', releases are cut from '$DEFAULT_BRANCH' only" >&2
  exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "release: the working tree is dirty, commit or stash first" >&2
  git status --short >&2
  exit 1
fi

if git rev-parse --verify --quiet "refs/tags/$TAG" >/dev/null; then
  echo "release: tag $TAG already exists locally, nothing was changed" >&2
  exit 1
fi

if [ -n "$(git ls-remote --tags origin "refs/tags/$TAG")" ]; then
  echo "release: tag $TAG already exists on origin, nothing was changed" >&2
  exit 1
fi

git fetch --quiet origin "$DEFAULT_BRANCH"
if [ "$(git rev-parse HEAD)" != "$(git rev-parse "origin/$DEFAULT_BRANCH")" ]; then
  echo "release: HEAD and origin/$DEFAULT_BRANCH differ, pull or push first" >&2
  exit 1
fi

if grep -Eq '^module .*/v[0-9]+$' go.mod; then
  echo "release: go.mod carries a /vN module path, which defeats the single-copy guarantee" >&2
  exit 1
fi

# --- build ------------------------------------------------------------------

echo "release: preparing $TAG"
go build ./...
go vet ./...
go test ./...

# --- tag, push --------------------------------------------------------------

# No manifest records the version, so there is nothing to commit: the tag alone
# is the release.
git tag -a "$TAG" -m "$TAG"
git push origin "$TAG"

echo "release: tagged $TAG; consumers pick it up with"
echo "  go get github.com/primegraph/primegraph-core-go@$TAG"
