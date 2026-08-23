# primegraph-core-go

`github.com/primegraph/primegraph-core-go` — the shared cross-package vocabulary for PrimeGraph
generated Go packages.

## Why this module exists

The PrimeGraph compiler generates one package per graph bucket. Each generated package used to carry
its own private copy of a runtime, so a type declared in that runtime existed once per package. When
one generated package handed a value to another — a model field, a returned error — the two copies
were different nominal types and the code broke: it failed to compile in Go and Swift, and in Kotlin a
`catch` silently failed to match.

This module holds the vocabulary that crosses package boundaries, so a graph has exactly one nominal
type per concept no matter how many generated packages it spans.

Per-bundle machinery — Firebase, HTTP transport, server helpers — stays inside the generated packages
and does **not** belong here.

There are five of these, one per target language:
`primegraph-core-ts`, `primegraph-core-go`, `primegraph-core-swift`, `primegraph-core-py`,
`primegraph-core-kt`.

## Status

Scaffolding only. No shared declarations have been migrated yet; `core.go` holds a single placeholder
so the build has something to compile.

## The module path never gains a `/vN` suffix

`github.com/primegraph/primegraph-core-go/v2` is, to the Go toolchain, a *different module* from
`github.com/primegraph/primegraph-core-go`. Both can be in one build graph at once, and Go will link
both into the same binary — reintroducing two nominal copies of every shared type, which is exactly
what this module exists to prevent.

So the module path stays as it is, permanently. Releases stay in the v0/v1 range; a breaking change is
coordinated across the generated packages instead of being escaped into a new major.
`scripts/release.sh` refuses a major of 2 or higher and refuses a `go.mod` whose module path ends in
`/vN`.

## Layout

```
go.mod              module declaration, no /vN suffix, ever
core.go             public surface
.githooks/          Conventional Commits hook, dependency-free POSIX shell
scripts/setup.sh    one-time clone setup
scripts/release.sh  the release procedure
```

## Setup

A fresh clone has to be pointed at the repository's own hooks once:

```sh
git config core.hooksPath .githooks
```

`sh scripts/setup.sh` does that for you.

The hook rejects any commit message that is not a Conventional Commit: `type(scope)!: subject`, one of
`build chore ci docs feat fix perf refactor revert style test`, header at most 100 characters, no
trailing period.

## Build

```sh
go build ./...
go vet ./...
```

## Releasing

```sh
sh scripts/release.sh 1.4.0
```

The argument is a bare semver — no leading `v`; the tag gets one. The script refuses to run on a dirty
tree, off the default branch, or when the tag already exists, and it does all of those checks before it
changes anything. Go records no version in a manifest and there is no registry to publish to, so the
release is tag-only: it builds, vets, tags and pushes the tag.

Consumers then pin it with `go get github.com/primegraph/primegraph-core-go@v1.4.0`.
