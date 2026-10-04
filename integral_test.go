package primegraphcore

import (
	"encoding/json"
	"testing"
)

// OpenAPI defines `integer` by value: `1.0` and `1e2` are integers, `1.5` is not.

func TestIntegralNumberRespellsWholeNumbers(t *testing.T) {
	cases := map[string]string{
		"1.0":                     "1",
		"1e2":                     "100",
		"1E+2":                    "100",
		"100.0e-2":                "1",
		"-2.50e1":                 "-25",
		"0.000":                   "0",
		"-0.0":                    "-0",
		"9.223372036854775807e18": "9223372036854775807",
		"-9223372036854775808.0":  "-9223372036854775808",
		// A float64 round-trip would land on 9007199254740992.
		"9007199254740993.0": "9007199254740993",
	}
	for in, want := range cases {
		if got := string(integralNumber([]byte(in))); got != want {
			t.Errorf("integralNumber(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestIntegralNumberKeepsEverythingElse(t *testing.T) {
	for _, in := range []string{
		"7", "-7", "1.5", "1e-1", "0.5", "2.000001",
		"9223372036854775808.0", "9.3e18", "1e19", "1e999999999", "1e99999999999999999999",
		`"1.0"`, "null", "true", "01.0", "1.", ".5", "+1.0", "1.0x",
	} {
		if got := string(integralNumber([]byte(in))); got != in {
			t.Errorf("integralNumber(%s) = %s, want it unchanged", in, got)
		}
	}
}

func TestNullableIntegersAcceptWholeNumerals(t *testing.T) {
	type model struct {
		Wide   NullableInt64 `json:"wide"`
		Narrow NullableInt32 `json:"narrow"`
		Plain  NullableInt   `json:"plain"`
	}
	var m model
	if err := json.Unmarshal([]byte(`{"wide":1.0,"narrow":1e2,"plain":-3.0e0}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if *m.Wide.Get() != 1 || *m.Narrow.Get() != 100 || *m.Plain.Get() != -3 {
		t.Fatalf("decoded %d / %d / %d", *m.Wide.Get(), *m.Narrow.Get(), *m.Plain.Get())
	}

	var exact NullableInt64
	if err := json.Unmarshal([]byte("9.223372036854775807e18"), &exact); err != nil {
		t.Fatalf("unmarshal max int64: %v", err)
	}
	if *exact.Get() != 9223372036854775807 {
		t.Fatalf("max int64 decoded as %d", *exact.Get())
	}
}

func TestNullableIntegersStillRefuseNonIntegers(t *testing.T) {
	for _, in := range []string{"1.5", "1e-1", "9.3e18", `"1"`} {
		var v NullableInt64
		if err := json.Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("NullableInt64 accepted %s as %d", in, *v.Get())
		}
	}
	var narrow NullableInt32
	if err := json.Unmarshal([]byte("3e9"), &narrow); err == nil {
		t.Errorf("NullableInt32 accepted 3e9 as %d", *narrow.Get())
	}
}

func TestNullableIntegerNullStaysPresentAndEmpty(t *testing.T) {
	var v NullableInt64
	if err := json.Unmarshal([]byte("null"), &v); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !v.IsSet() || v.Get() != nil {
		t.Fatalf("null: isSet=%v value=%v", v.IsSet(), v.Get())
	}
}
