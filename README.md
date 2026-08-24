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

## What lives here

Everything below crosses a package boundary: a value built in one generated package is read, matched
or marshalled in another.

| Declaration | Why it is shared |
| --- | --- |
| `NullableBool` `NullableInt` `NullableInt32` `NullableInt64` `NullableFloat32` `NullableFloat64` `NullableString` `NullableTime` `NullableDate`, their `Get`/`Set`/`IsSet`/`Unset`/`MarshalJSON`/`UnmarshalJSON`, and the nine `NewNullable*` constructors | the carrier of a `nullable: true` model field, so it appears in one package's model and another package's signature |
| `NullableStringOf` `NullableInt64Of` `NullableFloat64Of` `NullableBoolOf` | lift a scalar into that carrier; a package that calls one must resolve the type it returns |
| `Date` with `MarshalJSON`/`UnmarshalJSON` | the `format: date` carrier, and the inner type of `NullableDate` |
| `File`, `FormFile`, `TypeOf` | the DSL file value travels between packages as a parameter and a model field |
| `HttpAuth`, `HttpRequest`, `HttpResponse` | pure declaration with no behaviour, repeated verbatim by every package that emits an HTTP step; the `Fetch` that reads them stays generated, and a generated package aliases these instead of redeclaring them |
| `DslError[P]`, `NewDslError`, `DslErrorAny`, `DslErrorView[P]`, `DefaultErrorMessage`, `CoerceError`, `CoerceErrorDefault`, `MatchesDslError`, `MatchDslError`, `NewValidationError`, `ValidationMessage` | an error raised in one package is caught in another; `errors.As` matches on type identity, so a second declaration means a `catch` that never fires |
| `TransportErrorCoercer`, `FirebaseAdminErrorCoercer` | hook variables every package must read the *same* one of; the code that installs them stays in the generated packages, next to the SDK it names |

The package has no third-party dependencies. Everything here builds on the standard library alone.

## What does not live here

`MappedNullable`, `IsNil`, `NewStrictDecoder`, the `Ptr*` family, `NormalizeUUID*`, the schema and
validator machinery, the pure helpers whose signatures name only builtins, the HTTP server helpers,
the `Fetch` transport itself, and everything Firebase. None of them crosses a package boundary, and several would drag a
third-party module into every generated Go package.

Two things sit just outside the line and are worth naming:

- `grpcCodeToDslCode` stays in the generated packages. It is unexported and its only caller is the
  `init()` that installs `TransportErrorCoercer`, which stays there too; moving it would need
  `google.golang.org/grpc/codes` in the module graph of every generated Go package, including ones
  that never make a gRPC call. The *codes* it produces are this package's vocabulary, and
  `DefaultErrorMessage` answers for each of them — pinned by a test.
- `FileEqual` stays in the generated packages. It calls `BytesEqual`, a builtin-only helper that does
  not belong here, and being pure it has no nominal identity to share.

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

One Go package at the module root, named `primegraphcore`. Generated code imports it under an alias,
and a single package means a single import per generated file and one mechanical `X` → `alias.X`
rename in the emitter. Splitting the vocabulary across packages would buy nothing — the declarations
reference each other — and would cost the emitter a symbol-to-package map.

```
go.mod              module declaration, no /vN suffix, ever
doc.go              package documentation
nullable.go         the nine Nullable* carriers, their constructors and the *Of lifts
date.go             the calendar-day carrier
file.go             the file value, its multipart part, and the DSL type name of a value
http.go             the request and response value types of one outbound HTTP call
dslerror.go         the DslError family, the coercer hooks and the validation error
*_test.go           tests, co-located with what they cover
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

## Build and test

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

`gofmt -l .` must print nothing.

## Releasing

```sh
sh scripts/release.sh 1.4.0
```

The argument is a bare semver — no leading `v`; the tag gets one. The script refuses to run on a dirty
tree, off the default branch, or when the tag already exists, and it does all of those checks before it
changes anything. Go records no version in a manifest and there is no registry to publish to, so the
release is tag-only: it builds, vets, tests, tags and pushes the tag.

Consumers then pin it with `go get github.com/primegraph/primegraph-core-go@v1.4.0`.
