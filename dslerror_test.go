package primegraphcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"testing"
	"time"
)

// CoerceError logs a foreign error's own text rather than returning it, so the
// tests that walk that path silence the logger instead of dirtying the output.
func quietLog(t *testing.T) {
	t.Helper()
	previous := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(previous) })
}

// The coercer hooks are package-level by design — every generated package must
// read the same one — so a test that installs one puts it back.
func withTransportCoercer(t *testing.T, coercer func(error) (string, bool)) {
	t.Helper()
	previous := TransportErrorCoercer
	TransportErrorCoercer = coercer
	t.Cleanup(func() { TransportErrorCoercer = previous })
}

func withFirebaseAdminCoercer(t *testing.T, coercer func(error) (string, bool)) {
	t.Helper()
	previous := FirebaseAdminErrorCoercer
	FirebaseAdminErrorCoercer = coercer
	t.Cleanup(func() { FirebaseAdminErrorCoercer = previous })
}

func TestDslErrorConstruction(t *testing.T) {
	err := NewDslError("NOT_FOUND", map[string]any{"id": "42"})

	if err.Code != "NOT_FOUND" {
		t.Errorf("Code = %q, want NOT_FOUND", err.Code)
	}
	if err.GetCode() != "NOT_FOUND" {
		t.Errorf("GetCode = %q, want NOT_FOUND", err.GetCode())
	}
	if !reflect.DeepEqual(err.GetPayload(), map[string]any{"id": "42"}) {
		t.Errorf("GetPayload = %v", err.GetPayload())
	}
	if got, want := err.Error(), "DslError[NOT_FOUND]: map[id:42]"; got != want {
		t.Errorf("Error = %q, want %q", got, want)
	}

	var asError error = err
	var anyDsl DslErrorAny
	if !errors.As(asError, &anyDsl) {
		t.Fatal("a *DslError does not satisfy DslErrorAny")
	}
	if anyDsl.GetCode() != "NOT_FOUND" {
		t.Errorf("through DslErrorAny: GetCode = %q", anyDsl.GetCode())
	}
}

// The catch binding is what a DSL `catch` reads, and its JSON keys are the
// error shape a transport puts on the wire.
func TestDslErrorViewJSON(t *testing.T) {
	view := DslErrorView[map[string]any]{Code: "FORBIDDEN", Payload: map[string]any{"why": "scope"}}
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	const want = `{"code":"FORBIDDEN","payload":{"why":"scope"}}`
	if string(data) != want {
		t.Fatalf("marshal = %s, want %s", data, want)
	}
}

func TestDefaultErrorMessage(t *testing.T) {
	cases := map[string]string{
		"AUTH_REQUIRED":        "Authorization required",
		"VALIDATION_FAILED":    "Request validation failed",
		"NOT_FOUND":            "Resource not found",
		"already-exists":       "Resource already exists",
		"unauthenticated":      "Authorization required",
		"JSON_PARSE_FAILED":    "Input is not valid JSON",
		"NO_SUCH_CODE_AT_ALL":  "Internal server error",
		"":                     "Internal server error",
		"DECIMAL_PARSE_FAILED": "Input is not a decimal number",
	}
	for code, want := range cases {
		if got := DefaultErrorMessage(code); got != want {
			t.Errorf("DefaultErrorMessage(%q) = %q, want %q", code, got, want)
		}
	}
}

// grpcDslCodes is the DSL half of the gRPC status mapping a generated package
// installs on TransportErrorCoercer. The mapping function itself needs
// google.golang.org/grpc/codes and stays in the generated package; the code
// vocabulary it produces is this package's, so this package owes each of them a
// default message.
var grpcDslCodes = []string{
	"cancelled",
	"unknown",
	"invalid-argument",
	"deadline-exceeded",
	"not-found",
	"already-exists",
	"permission-denied",
	"resource-exhausted",
	"failed-precondition",
	"aborted",
	"out-of-range",
	"internal",
	"unavailable",
	"data-loss",
	"unauthenticated",
}

func TestEveryTransportCodeHasADefaultMessage(t *testing.T) {
	for _, code := range grpcDslCodes {
		if _, known := dslErrorMessages[code]; !known {
			t.Errorf("no default message for transport code %q", code)
		}
	}
}

func TestCoerceErrorKeepsATypedRaise(t *testing.T) {
	type payload struct {
		Reason string `json:"reason"`
	}
	err := NewDslError("FORBIDDEN", payload{Reason: "no scope"})

	got := CoerceError[payload](err, payload{Reason: "fallback"})
	if got.Code != "FORBIDDEN" || got.Payload.Reason != "no scope" {
		t.Fatalf("CoerceError = %+v, want the raised code and payload", got)
	}
}

// A raise reaches a catch through errors.As only when the catch declares the
// same instantiation. Any other one goes through DslErrorAny, which keeps the
// code and hands back the payload when it happens to fit.
func TestCoerceErrorThroughDslErrorAny(t *testing.T) {
	err := NewDslError("NOT_FOUND", "the thing")

	// T = any: the payload always fits.
	loose := CoerceError[any](err, nil)
	if loose.Code != "NOT_FOUND" || loose.Payload != "the thing" {
		t.Fatalf("CoerceError[any] = %+v", loose)
	}

	// T = a shape the payload is not: the catch's own default stands in.
	mismatched := CoerceError[map[string]any](err, map[string]any{"d": true})
	if mismatched.Code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", mismatched.Code)
	}
	if !reflect.DeepEqual(mismatched.Payload, map[string]any{"d": true}) {
		t.Fatalf("payload = %v, want the declared default", mismatched.Payload)
	}
}

// A text payload slot the catch leaves empty is filled from the code's default
// message, so an error view always carries a code AND a message.
func TestCoerceErrorFillsAnEmptyTextPayload(t *testing.T) {
	err := NewDslError("NOT_FOUND", 404)

	filled := CoerceError[string](err, "")
	if filled.Code != "NOT_FOUND" || filled.Payload != "Resource not found" {
		t.Fatalf("CoerceError = %+v, want the default message", filled)
	}

	// A non-empty default is the author's own text and is left alone.
	kept := CoerceError[string](err, "mine")
	if kept.Payload != "mine" {
		t.Fatalf("payload = %q, want the declared default", kept.Payload)
	}
}

func TestCoerceErrorForeignError(t *testing.T) {
	quietLog(t)
	withTransportCoercer(t, nil)

	got := CoerceError[string](errors.New("connection reset"), "")
	if got.Code != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", got.Code)
	}
	if got.Payload != "Internal server error" {
		t.Fatalf("payload = %q, want the default message", got.Payload)
	}
	// The foreign error's own wording never reaches the view.
	if got.Payload == "connection reset" {
		t.Fatal("the foreign error text leaked into the view")
	}
}

// The whole point of the hook: a raw SDK failure surfaces on a catch as the
// matching DSL code instead of collapsing to INTERNAL_ERROR.
func TestCoerceErrorThroughTheTransportCoercer(t *testing.T) {
	quietLog(t)

	type sdkError struct {
		error
		code string
	}
	withTransportCoercer(t, func(err error) (string, bool) {
		var typed sdkError
		if errors.As(err, &typed) {
			return typed.code, true
		}
		return "", false
	})

	for _, code := range grpcDslCodes {
		raw := sdkError{error: errors.New("raw sdk failure"), code: code}
		got := CoerceError[string](raw, "")
		if got.Code != code {
			t.Errorf("code = %q, want %q", got.Code, code)
		}
		if got.Payload != DefaultErrorMessage(code) {
			t.Errorf("payload for %q = %q, want %q", code, got.Payload, DefaultErrorMessage(code))
		}
	}

	// An error the coercer does not name still falls back.
	if got := CoerceError[string](errors.New("nameless"), ""); got.Code != "INTERNAL_ERROR" {
		t.Errorf("unnamed error code = %q, want INTERNAL_ERROR", got.Code)
	}
}

// A DSL raise names itself; the coercer is only consulted for what it does not.
func TestCoerceErrorPrefersADslRaiseOverTheCoercer(t *testing.T) {
	withTransportCoercer(t, func(error) (string, bool) { return "unavailable", true })

	got := CoerceError[string](NewDslError("FORBIDDEN", "denied"), "")
	if got.Code != "FORBIDDEN" {
		t.Fatalf("code = %q, want FORBIDDEN — the raise, not the coercer", got.Code)
	}
}

// The generated firebase_admin.go installs its naming on the second hook, which
// the transport coercer consults after the gRPC branch. Only the variable lives
// here, so what is pinned is that the variable is shared and readable.
func TestFirebaseAdminCoercerHookIsShared(t *testing.T) {
	quietLog(t)
	withFirebaseAdminCoercer(t, func(error) (string, bool) { return "already-exists", true })
	withTransportCoercer(t, func(err error) (string, bool) {
		if FirebaseAdminErrorCoercer != nil {
			if mapped, ok := FirebaseAdminErrorCoercer(err); ok {
				return mapped, true
			}
		}
		return "", false
	})

	if got := CoerceError[string](errors.New("admin failure"), ""); got.Code != "already-exists" {
		t.Fatalf("code = %q, want already-exists", got.Code)
	}
}

func TestCoerceErrorDefaultUsesTheZeroValue(t *testing.T) {
	quietLog(t)
	withTransportCoercer(t, nil)

	type payload struct {
		Reason string `json:"reason"`
		Count  int    `json:"count"`
	}
	got := CoerceErrorDefault[payload](errors.New("boom"))
	if got.Code != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", got.Code)
	}
	if got.Payload != (payload{}) {
		t.Fatalf("payload = %+v, want the zero value", got.Payload)
	}

	// The zero of a text payload is empty, so it is filled like any other.
	if text := CoerceErrorDefault[string](errors.New("boom")); text.Payload != "Internal server error" {
		t.Fatalf("text payload = %q, want the default message", text.Payload)
	}
}

func TestMatchesDslError(t *testing.T) {
	quietLog(t)
	withTransportCoercer(t, nil)

	requiresReason := func(v any) error {
		obj, ok := v.(map[string]any)
		if !ok {
			return errors.New("not an object")
		}
		if _, present := obj["reason"]; !present {
			return errors.New("no reason")
		}
		return nil
	}

	matching := NewDslError("FORBIDDEN", map[string]any{"reason": "scope"})
	if !MatchesDslError(matching, requiresReason) {
		t.Error("a payload the catch declares did not match")
	}

	other := NewDslError("FORBIDDEN", map[string]any{"other": "thing"})
	if MatchesDslError(other, requiresReason) {
		t.Error("a payload the catch does not declare matched")
	}

	// A catch that asserts nothing checkable takes whatever arrived.
	if !MatchesDslError(other, nil) {
		t.Error("a catch with no predicate refused an arrived payload")
	}

	// A foreign error contributes the standard {code, message} object, so a
	// catch declaring that shape handles it and one declaring anything else
	// does not.
	requiresCodeAndMessage := func(v any) error {
		obj, ok := v.(map[string]any)
		if !ok {
			return errors.New("not an object")
		}
		if _, hasCode := obj["code"]; !hasCode {
			return errors.New("no code")
		}
		if _, hasMessage := obj["message"]; !hasMessage {
			return errors.New("no message")
		}
		return nil
	}
	foreign := errors.New("connection reset")
	if !MatchesDslError(foreign, requiresCodeAndMessage) {
		t.Error("a foreign error did not present the standard error object")
	}
	if MatchesDslError(foreign, requiresReason) {
		t.Error("a foreign error matched a catch declaring an unrelated shape")
	}
}

// Nil when the value is not JSON at all, which no schema could accept either.
func TestMatchDslErrorRejectsAnUnmarshallablePayload(t *testing.T) {
	raised := NewDslError("FORBIDDEN", make(chan int))

	if MatchesDslError(raised, nil) {
		t.Error("a payload that cannot be marshalled matched")
	}
	if _, ok := MatchDslError[any](raised, nil); ok {
		t.Error("a payload that cannot be marshalled bound a value")
	}
}

func TestMatchDslErrorBindsTheDecodedPayload(t *testing.T) {
	type binding struct {
		Reason string    `json:"reason"`
		At     time.Time `json:"at"`
	}
	accepts := func(v any) error {
		if _, ok := v.(map[string]any); !ok {
			return errors.New("not an object")
		}
		return nil
	}

	raised := NewDslError("FORBIDDEN", map[string]any{
		"reason": "scope",
		"at":     "2024-03-05T06:07:08Z",
	})
	got, ok := MatchDslError[binding](raised, accepts)
	if !ok {
		t.Fatal("the catch did not match")
	}
	if got.Code != "FORBIDDEN" || got.Payload.Reason != "scope" {
		t.Fatalf("binding = %+v", got)
	}
	// A date-time slot arrives as the instant it names, not as its text.
	if !got.Payload.At.Equal(time.Date(2024, 3, 5, 6, 7, 8, 0, time.UTC)) {
		t.Fatalf("at = %s, want the parsed instant", got.Payload.At)
	}

	refused := func(any) error { return errors.New("no") }
	if _, ok := MatchDslError[binding](raised, refused); ok {
		t.Error("a refusing predicate still matched")
	}

	// A payload the declared type cannot decode is not a match either.
	notAnObject := NewDslError("FORBIDDEN", "plain text")
	if _, ok := MatchDslError[binding](notAnObject, nil); ok {
		t.Error("a payload the binding cannot decode matched")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("body/email", "string", "number")

	var typed *DslError[map[string]any]
	if !errors.As(err, &typed) {
		t.Fatal("NewValidationError did not produce a *DslError[map[string]any]")
	}
	if typed.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", typed.Code)
	}
	want := map[string]any{"path": "body/email", "expected": "string", "actual": "number"}
	if !reflect.DeepEqual(typed.Payload, want) {
		t.Errorf("payload = %v, want %v", typed.Payload, want)
	}
}

// ValidationMessage degrading to its generic fallback is the bug this package
// exists to fix: it resolves the error through errors.As on
// *DslError[map[string]any], which only succeeds when the raiser and the reader
// name one and the same type. With a copy of DslError per generated package
// they never did, and every cross-package validation failure came out as
// "Request validation failed".
func TestValidationMessageRendersTheSpecificFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "a named path",
			err:  NewValidationError("body/email", "string", "number"),
			want: "body/email: expected string, got number",
		},
		{
			name: "an empty path names the root",
			err:  NewValidationError("", "object", "array"),
			want: "(root): expected object, got array",
		},
		{
			name: "a nested path is kept whole",
			err:  NewValidationError("body/items/3/id", "format uuid", "not-a-uuid"),
			want: "body/items/3/id: expected format uuid, got not-a-uuid",
		},
		{
			name: "a wrapped error is still resolved",
			err:  fmt.Errorf("decoding the request: %w", NewValidationError("query/limit", "integer", "number")),
			want: "query/limit: expected integer, got number",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValidationMessage(c.err); got != c.want {
				t.Fatalf("ValidationMessage = %q, want %q", got, c.want)
			}
		})
	}
}

func TestValidationMessageFallsBackOnlyForNonValidationErrors(t *testing.T) {
	const generic = "Request validation failed"
	cases := []struct {
		name string
		err  error
	}{
		{"a plain error", errors.New("connection reset")},
		{"no error at all", nil},
		{"a DSL raise carrying another payload type", NewDslError("VALIDATION_FAILED", "text")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValidationMessage(c.err); got != generic {
				t.Fatalf("ValidationMessage = %q, want the generic fallback", got)
			}
		})
	}
}

// foreignDslError has the same fields and the same methods as
// *DslError[map[string]any] and is still a different type. It stands in for the
// per-package copy that used to exist: structural sameness buys nothing,
// errors.As matches on identity, so the reader must be looking at the same
// declaration the raiser used. That is the whole argument for this package.
type foreignDslError struct {
	Code    string
	Payload map[string]any
}

func (e *foreignDslError) Error() string      { return "DslError[" + e.Code + "]" }
func (e *foreignDslError) GetCode() string    { return e.Code }
func (e *foreignDslError) GetPayload() any    { return e.Payload }
func (e *foreignDslError) Unwrap() error      { return nil }
func (e *foreignDslError) isValidation() bool { return e.Code == "VALIDATION_FAILED" }

func TestValidationMessageCannotReadADuplicateDeclaration(t *testing.T) {
	duplicate := &foreignDslError{
		Code:    "VALIDATION_FAILED",
		Payload: map[string]any{"path": "body/email", "expected": "string", "actual": "number"},
	}
	if !duplicate.isValidation() {
		t.Fatal("the stand-in is not shaped like a validation error")
	}
	if got := ValidationMessage(duplicate); got != "Request validation failed" {
		t.Fatalf("ValidationMessage = %q; a second declaration must not resolve", got)
	}
	// And the one true declaration does resolve — same payload, same shape.
	if got := ValidationMessage(NewValidationError("body/email", "string", "number")); got != "body/email: expected string, got number" {
		t.Fatalf("ValidationMessage = %q, want the specific line", got)
	}
}
