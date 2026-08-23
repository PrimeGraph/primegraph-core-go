package primegraphcore

import (
	"encoding/json"
	"testing"
	"time"
)

// The wire form of a file is the contract every target shares: `name`,
// `mimeType`, and the bytes as base64 under `data`.
func TestFileJSONShape(t *testing.T) {
	f := File{Name: "invoice.pdf", Type: "application/pdf", Data: []byte("hi")}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	const want = `{"name":"invoice.pdf","mimeType":"application/pdf","data":"aGk="}`
	if string(data) != want {
		t.Fatalf("marshal = %s, want %s", data, want)
	}

	var back File
	if err := json.Unmarshal([]byte(want), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Name != f.Name || back.Type != f.Type || string(back.Data) != string(f.Data) {
		t.Fatalf("round trip = %+v, want %+v", back, f)
	}
}

func TestFormFile(t *testing.T) {
	cases := []struct {
		name  string
		field string
		file  File
		want  string
	}{
		{
			name:  "declared content type",
			field: "upload",
			file:  File{Name: "a.txt", Type: "text/plain", Data: []byte("body")},
			want: "Content-Disposition: form-data; name=\"upload\"; filename=\"a.txt\"\r\n" +
				"Content-Type: text/plain\r\n\r\nbody",
		},
		{
			name:  "absent content type falls back to octet-stream",
			field: "upload",
			file:  File{Name: "a.bin", Data: []byte{0x00, 0x01}},
			want: "Content-Disposition: form-data; name=\"upload\"; filename=\"a.bin\"\r\n" +
				"Content-Type: application/octet-stream\r\n\r\n\x00\x01",
		},
		{
			// %q on the name and filename is what keeps a quote in either one from
			// ending the header field it sits in.
			name:  "quotes in the names are escaped",
			field: `od"d`,
			file:  File{Name: `we"ird.txt`, Type: "text/plain", Data: nil},
			want: "Content-Disposition: form-data; name=\"od\\\"d\"; filename=\"we\\\"ird.txt\"\r\n" +
				"Content-Type: text/plain\r\n\r\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(FormFile(c.field, c.file))
			if got != c.want {
				t.Fatalf("FormFile = %q, want %q", got, c.want)
			}
		})
	}
}

func TestTypeOf(t *testing.T) {
	cases := []struct {
		value any
		want  string
	}{
		{nil, "null"},
		{true, "bool"},
		{"text", "string"},
		{[]byte("raw"), "bytes"},
		{File{Name: "a"}, "file"},
		{int(1), "int"},
		{int8(1), "int"},
		{int16(1), "int"},
		{int32(1), "int"},
		{int64(1), "int"},
		{float32(1.5), "double"},
		{float64(1.5), "double"},
		{[]any{1, 2}, "list"},
		{[]string{"a"}, "list"},
		{[2]int{1, 2}, "list"},
		{map[string]any{"a": 1}, "object"},
		{struct{ A int }{1}, "object"},
		{time.Time{}, "object"},
		// Nothing the DSL can name reaches here; the vocabulary has no third
		// answer, so an unnameable value is an object.
		{make(chan int), "object"},
		{func() {}, "object"},
		{new(int), "object"},
	}
	for _, c := range cases {
		if got := TypeOf(c.value); got != c.want {
			t.Errorf("TypeOf(%#v) = %q, want %q", c.value, got, c.want)
		}
	}
}
