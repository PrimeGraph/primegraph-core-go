package primegraphcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

type DslError[Payload any] struct {
	Code    string
	Payload Payload
}

func (e *DslError[Payload]) Error() string {
	return fmt.Sprintf("DslError[%s]: %v", e.Code, e.Payload)
}

func (e *DslError[Payload]) GetCode() string {
	return e.Code
}

func (e *DslError[Payload]) GetPayload() any {
	return e.Payload
}

func NewDslError[Payload any](code string, payload Payload) *DslError[Payload] {
	return &DslError[Payload]{Code: code, Payload: payload}
}

type DslErrorAny interface {
	error
	GetCode() string
	GetPayload() any
}

type DslErrorView[Payload any] struct {
	Code    string  `json:"code"`
	Payload Payload `json:"payload"`
}

// Default text of each error code. A coerced error has no message of its own —
// a foreign SDK's wording never reaches a client — so a text payload slot it
// leaves empty is filled from here, and the error view always carries a code
// AND a message.
var dslErrorMessages = map[string]string{
	"AUTH_REQUIRED":           "Authorization required",
	"VALIDATION_FAILED":       "Request validation failed",
	"INTERNAL_ERROR":          "Internal server error",
	"NOT_FOUND":               "Resource not found",
	"METHOD_NOT_ALLOWED":      "Method not allowed",
	"FORBIDDEN":               "Access forbidden",
	"NO_RESPONSE":             "Block did not produce a response",
	"invalid-argument":        "Invalid argument",
	"failed-precondition":     "Failed precondition",
	"out-of-range":            "Value out of range",
	"unauthenticated":         "Authorization required",
	"permission-denied":       "Access forbidden",
	"not-found":               "Resource not found",
	"already-exists":          "Resource already exists",
	"resource-exhausted":      "Resource exhausted",
	"cancelled":               "Request cancelled",
	"data-loss":               "Data loss",
	"unknown":                 "Unknown error",
	"internal":                "Internal server error",
	"unavailable":             "Service unavailable",
	"deadline-exceeded":       "Deadline exceeded",
	"aborted":                 "Operation aborted",
	"UNSUPPORTED_MEDIA_TYPE":  "Unsupported media type",
	"UTF8_DECODE_FAILED":      "Input is not valid UTF-8",
	"BASE64_DECODE_FAILED":    "Input is not valid base64",
	"HEX_DECODE_FAILED":       "Input is not valid hex",
	"URL_DECODE_FAILED":       "Input is not valid percent-encoded text",
	"ENUM_VALUE_NOT_A_MEMBER": "Value is not a member of the enumeration",
	"JSON_PARSE_FAILED":       "Input is not valid JSON",
	"DECIMAL_PARSE_FAILED":    "Input is not a decimal number",
	"DURATION_PARSE_FAILED":   "Input is not a valid duration",
	"TIME_PARSE_FAILED":       "Input does not match the expected time format",
}

func DefaultErrorMessage(code string) string {
	if msg, ok := dslErrorMessages[code]; ok {
		return msg
	}
	return "Internal server error"
}

func coercedPayload[T any](code string, defaultPayload T) T {
	if text, isText := any(defaultPayload).(string); isText && text == "" {
		if filled, ok := any(DefaultErrorMessage(code)).(T); ok {
			return filled
		}
	}
	return defaultPayload
}

// TransportErrorCoercer, when installed by a transport runtime (e.g. the
// Firestore/gRPC helper), maps a raw SDK error onto a DSL code so a broad catch
// can match it. The generic runtime carries no transport dependency; a
// transport helper registers this hook from its own init() when present.
var TransportErrorCoercer func(error) (string, bool)

// arrivedErrorCode is the DSL code a caught error presents: its own when it is
// a DSL raise, the code the installed coercer maps it to when it is a
// transport/SDK failure, INTERNAL_ERROR when nothing names it.
func arrivedErrorCode(err error) string {
	var anyDsl DslErrorAny
	if errors.As(err, &anyDsl) {
		return anyDsl.GetCode()
	}
	if TransportErrorCoercer != nil {
		if mapped, ok := TransportErrorCoercer(err); ok {
			return mapped
		}
	}
	return "INTERNAL_ERROR"
}

// CoerceError projects ANY caught error onto the catch binding {Code, Payload}.
// A DSL raise yields its own code, plus its payload whenever that payload fits
// the type the catch declares — where it does not, that type's default stands
// in. A transport/SDK failure (Firestore, gRPC, ...) is normalized into the DSL
// code space via the installed TransportErrorCoercer, so a catch on
// err.Code == "already-exists" matches. Anything else becomes code
// "INTERNAL_ERROR" with the default payload, so a broad catch always hands the
// body a well-shaped {Code, Payload} value.
func CoerceError[T any](err error, defaultPayload T) DslErrorView[T] {
	var typed *DslError[T]
	if errors.As(err, &typed) && typed != nil {
		return DslErrorView[T]{Code: typed.Code, Payload: typed.Payload}
	}
	var anyDsl DslErrorAny
	if errors.As(err, &anyDsl) {
		// The typed match above only sees the one instantiation the catch
		// declares, so a raise carrying any other payload type lands here — a
		// oneOf catch, whose T is `any`, always does.
		dslCode := anyDsl.GetCode()
		if payload, ok := anyDsl.GetPayload().(T); ok {
			return DslErrorView[T]{Code: dslCode, Payload: payload}
		}
		return DslErrorView[T]{Code: dslCode, Payload: coercedPayload(dslCode, defaultPayload)}
	}
	code := arrivedErrorCode(err)
	// A foreign error carries no typed payload, so its own text would be lost
	// here. It goes to the log instead of the returned view: the code is what a
	// caller may put on the wire, the text stays server-side.
	log.Printf("[dsl] error coerced to %s: %v", code, err)
	return DslErrorView[T]{Code: code, Payload: coercedPayload(code, defaultPayload)}
}

// CoerceErrorDefault is CoerceError with the zero value of the catch binding's
// declared payload type as the default, so each catch site starts from its own
// declared default instead of whatever an earlier catch left in the scope slot.
func CoerceErrorDefault[T any](err error) DslErrorView[T] {
	var zero T
	return CoerceError(err, zero)
}

// arrivedErrorPayload renders what arrived with a caught error as the JSON tree
// a declared schema describes. A DSL raise contributes its payload's wire form.
// A foreign error carries no payload of its own, so it contributes the standard
// error object built from the code it maps to — a catch declaring that shape
// handles it, a catch declaring anything else does not. Nil when the value is
// not JSON at all, which no schema could accept either.
func arrivedErrorPayload(err error) []byte {
	var anyDsl DslErrorAny
	if errors.As(err, &anyDsl) {
		data, marshalErr := json.Marshal(anyDsl.GetPayload())
		if marshalErr != nil {
			return nil
		}
		return data
	}
	code := arrivedErrorCode(err)
	data, marshalErr := json.Marshal(map[string]string{
		"code":    code,
		"message": DefaultErrorMessage(code),
	})
	if marshalErr != nil {
		return nil
	}
	return data
}

// acceptedArrivedError is the arrived payload's JSON when the shape a catch
// declares accepts it, nil when it does not. A catch whose declared type
// asserts nothing checkable passes no predicate and takes whatever arrived.
func acceptedArrivedError(err error, accepts func(any) error) []byte {
	data := arrivedErrorPayload(err)
	if data == nil {
		return nil
	}
	if accepts == nil {
		return data
	}
	var decoded any
	if json.Unmarshal(data, &decoded) != nil {
		return nil
	}
	if accepts(decoded) != nil {
		return nil
	}
	return data
}

// MatchesDslError reports whether a catch handles the caught error. A catch
// handles it exactly when the payload that arrived validates against the shape
// the catch declares — never by the payload's Go type. Used where the catch
// body reads no binding: the match still decides whether that body runs.
func MatchesDslError(err error, accepts func(any) error) bool {
	return acceptedArrivedError(err, accepts) != nil
}

// MatchDslError is MatchesDslError plus the value the catch binds. The arrived
// payload is decoded as the type the catch declares, so a date-time slot
// arrives as the instant it names rather than as the text it travelled as.
func MatchDslError[T any](err error, accepts func(any) error) (DslErrorView[T], bool) {
	data := acceptedArrivedError(err, accepts)
	if data == nil {
		return DslErrorView[T]{}, false
	}
	var payload T
	if json.Unmarshal(data, &payload) != nil {
		return DslErrorView[T]{}, false
	}
	return DslErrorView[T]{Code: arrivedErrorCode(err), Payload: payload}, true
}

// FirebaseAdminErrorCoercer names a firebase-admin failure in the DSL code
// space. An Admin SDK error is not a gRPC status — internal.FirebaseError
// carries a string code the SDK exposes through errorutils — and reading it
// needs the firebase module, which only a bundle that calls firebase has. So
// runtime/firebase_admin.go installs this hook from its own init() and a bundle
// without it keeps the shared runtime free of the dependency.
var FirebaseAdminErrorCoercer func(error) (string, bool)

// NewValidationError builds the DslError every declared-type read boundary
// raises. The payload shape is fixed so a DSL catch can branch on it.
func NewValidationError(path string, expected string, actual string) error {
	return NewDslError[map[string]any]("VALIDATION_FAILED", map[string]any{
		"path":     path,
		"expected": expected,
		"actual":   actual,
	})
}

// ValidationMessage renders a validation failure as one readable line. A
// transport answering with an HTTP status needs text, not the payload map.
func ValidationMessage(err error) string {
	var typed *DslError[map[string]any]
	if !errors.As(err, &typed) || typed == nil {
		return "Request validation failed"
	}
	path, _ := typed.Payload["path"].(string)
	expected, _ := typed.Payload["expected"].(string)
	actual, _ := typed.Payload["actual"].(string)
	if path == "" {
		path = "(root)"
	}
	return fmt.Sprintf("%s: expected %s, got %s", path, expected, actual)
}
