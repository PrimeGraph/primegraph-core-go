package primegraphcore

import (
	"bytes"
	"fmt"
	"reflect"
)

// File is the DSL file value. The field names are Go's business; the tags are
// the contract. Every target puts the same object on the wire —
// `{name, mimeType, data}` with the bytes as base64 — and `encoding/json`
// writes a `[]byte` as base64 on its own.
type File struct {
	Name string `json:"name"`
	Type string `json:"mimeType"`
	Data []byte `json:"data"`
}

// FormFile renders one multipart part for a file field.
func FormFile(name string, file File) []byte {
	contentType := file.Type
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header := fmt.Sprintf("Content-Disposition: form-data; name=%q; filename=%q\r\nContent-Type: %s\r\n\r\n", name, file.Name, contentType)
	var buf bytes.Buffer
	buf.WriteString(header)
	buf.Write(file.Data)
	return buf.Bytes()
}

// TypeOf names a runtime value in the DSL type vocabulary.
func TypeOf(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case bool:
		return "bool"
	case string:
		return "string"
	case []byte:
		return "bytes"
	case File:
		return "file"
	}
	k := reflect.TypeOf(v).Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.Slice, reflect.Array:
		return "list"
	case reflect.Map, reflect.Struct:
		return "object"
	}
	return "object"
}
