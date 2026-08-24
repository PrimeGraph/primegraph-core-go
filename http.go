package primegraphcore

// The shape of one outbound HTTP call and of what it returned.
//
// Only the value types live here. The transport that reads them — `Fetch`, the
// auth application, the response decode — is per-bundle machinery and stays in
// the generated package, which is why these three carry no behaviour: they were
// repeated verbatim in every generated package that emits an HTTP step, and a
// generated package now aliases them instead of redeclaring them.

// HttpAuth is the credential one HTTP step applies. `Type` selects which of the
// remaining fields carries the secret; the unused ones stay empty rather than
// becoming a pointer each, because the value is built in one literal at the
// call site.
type HttpAuth struct {
	Type     string
	Scheme   string
	In       string
	Name     string
	Value    string
	Username string
	Password string
	Token    string
}

// HttpRequest is one outbound call. `Body` is already encoded: the call site
// owns the JSON / UTF-8 encoding, so the source-level expression type decides
// the Content-Type where it is still known. A nil `Timeout` means the step
// declared none and the call is unbounded.
type HttpRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Query   map[string]string
	Body    []byte
	Auth    *HttpAuth
	Timeout *float64
}

// HttpResponse is what one call returned. `Body` is the bytes exactly as they
// arrived — a typed body is what the step's declared response schemas are for.
// Header keys are lowercased by the transport, so a lookup needs no case walk.
type HttpResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
}
