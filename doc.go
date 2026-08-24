// Package primegraphcore holds the shared cross-package vocabulary for
// PrimeGraph generated Go packages, so that a graph has exactly one nominal
// type per concept no matter how many generated packages it spans.
//
// Everything declared here crosses a package boundary: a value built in one
// generated package is read, matched or marshalled in another. A copy per
// package would make those two values different nominal types — which is how a
// NullableString went undefined across packages and how a validation error
// stopped matching its own errors.As target.
//
// Per-bundle machinery — Firebase, the HTTP transport itself, server helpers,
// the schema validator, the pure builtin-only helpers — stays inside the
// generated packages and does not belong here. The HTTP request and response
// VALUE types are the exception the split makes obvious: they carry no
// behaviour and every package repeated them verbatim, so they are declared once
// here while the Fetch that reads them stays generated.
package primegraphcore
