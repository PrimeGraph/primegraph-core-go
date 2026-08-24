package primegraphcore

import (
	"reflect"
	"testing"
)

// The emitted call sites spell these field names literally — a step writes
// `runtime.HttpRequest{URL: ..., Method: ...}` — so the names and their types
// are the contract, not an implementation detail. Pinning them here turns a
// rename into a failing test in this module rather than into a compile error in
// every generated package.
func TestHttpTypeFields(t *testing.T) {
	cases := []struct {
		name   string
		value  any
		fields map[string]string
	}{
		{
			name:  "HttpAuth",
			value: HttpAuth{},
			fields: map[string]string{
				"Type":     "string",
				"Scheme":   "string",
				"In":       "string",
				"Name":     "string",
				"Value":    "string",
				"Username": "string",
				"Password": "string",
				"Token":    "string",
			},
		},
		{
			name:  "HttpRequest",
			value: HttpRequest{},
			fields: map[string]string{
				"URL":     "string",
				"Method":  "string",
				"Headers": "map[string]string",
				"Query":   "map[string]string",
				"Body":    "[]uint8",
				"Auth":    "*primegraphcore.HttpAuth",
				"Timeout": "*float64",
			},
		},
		{
			name:  "HttpResponse",
			value: HttpResponse{},
			fields: map[string]string{
				"Status":  "int",
				"Headers": "map[string]string",
				"Body":    "[]uint8",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			typ := reflect.TypeOf(c.value)
			if typ.NumField() != len(c.fields) {
				t.Fatalf("%s has %d fields, want %d", c.name, typ.NumField(), len(c.fields))
			}
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				want, ok := c.fields[field.Name]
				if !ok {
					t.Fatalf("%s has an undeclared field %q", c.name, field.Name)
				}
				if got := field.Type.String(); got != want {
					t.Errorf("%s.%s is %s, want %s", c.name, field.Name, got, want)
				}
				if !field.IsExported() {
					t.Errorf("%s.%s is unexported, a generated package could not name it", c.name, field.Name)
				}
			}
		})
	}
}

// A step that declares no auth and no timeout builds the request without them,
// so the zero value has to mean exactly that. `Timeout` is a pointer for the
// same reason: a declared zero is a bounded call, an absent one is not.
func TestHttpRequestZeroValueDeclaresNothing(t *testing.T) {
	var req HttpRequest
	if req.Auth != nil {
		t.Errorf("zero Auth = %v, want nil", req.Auth)
	}
	if req.Timeout != nil {
		t.Errorf("zero Timeout = %v, want nil", req.Timeout)
	}
	if req.Body != nil {
		t.Errorf("zero Body = %v, want nil", req.Body)
	}
	if len(req.Headers) != 0 || len(req.Query) != 0 {
		t.Errorf("zero Headers/Query = %v/%v, want empty", req.Headers, req.Query)
	}

	zero := 0.0
	req.Timeout = &zero
	if req.Timeout == nil || *req.Timeout != 0 {
		t.Errorf("a declared zero timeout must stay distinguishable from an absent one")
	}
}

// The transport lowercases the response header keys, so a generated package
// reads them without a case walk. Nothing here does the lowercasing — the point
// is that the map is a plain one and carries whatever the transport put in it.
func TestHttpResponseCarriesWhatItWasGiven(t *testing.T) {
	resp := HttpResponse{
		Status:  201,
		Headers: map[string]string{"content-type": "application/json"},
		Body:    []byte(`{"ok":true}`),
	}
	if resp.Status != 201 {
		t.Errorf("Status = %d, want 201", resp.Status)
	}
	if resp.Headers["content-type"] != "application/json" {
		t.Errorf("Headers = %v, want the content type it was given", resp.Headers)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("Body = %s, want the bytes it was given", resp.Body)
	}
}
