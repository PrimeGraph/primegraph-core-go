package primegraphcore

import (
	"encoding/json"
	"testing"
	"time"
)

// nullablePtr is the pointer half of a Nullable* wrapper: Set/Unset/UnmarshalJSON
// take a pointer receiver, Get/IsSet/MarshalJSON are promoted from the value.
type nullablePtr[T any, W any] interface {
	*W
	Set(*T)
	Unset()
	Get() *T
	IsSet() bool
	json.Marshaler
	json.Unmarshaler
}

// roundTripNullable exercises the full contract of one wrapper: the zero value
// is absent, Set makes it present, marshalling writes the bare value (not an
// object), unmarshalling a value and unmarshalling null both mark it present,
// and Unset returns it to absent.
func roundTripNullable[T comparable, W any, PW nullablePtr[T, W]](
	t *testing.T,
	name string,
	value T,
	wantJSON string,
) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var w W
		v := PW(&w)

		if v.IsSet() {
			t.Fatalf("%s: zero value reports IsSet", name)
		}
		if v.Get() != nil {
			t.Fatalf("%s: zero value carries a value", name)
		}

		v.Set(&value)
		if !v.IsSet() {
			t.Fatalf("%s: not set after Set", name)
		}
		if got := v.Get(); got == nil || *got != value {
			t.Fatalf("%s: Get = %v, want %v", name, got, value)
		}

		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if string(data) != wantJSON {
			t.Fatalf("%s: marshal = %s, want %s", name, data, wantJSON)
		}

		var back W
		bv := PW(&back)
		if err := json.Unmarshal(data, bv); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
		if !bv.IsSet() {
			t.Fatalf("%s: not set after unmarshal", name)
		}
		if got := bv.Get(); got == nil || *got != value {
			t.Fatalf("%s: round trip = %v, want %v", name, got, value)
		}

		// Present-and-null is a state of its own: it marshals to null and
		// survives the trip back as "set, with no value".
		v.Set(nil)
		data, err = json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: marshal nil: %v", name, err)
		}
		if string(data) != "null" {
			t.Fatalf("%s: marshal nil = %s, want null", name, data)
		}

		var nullBack W
		nv := PW(&nullBack)
		if err := json.Unmarshal([]byte("null"), nv); err != nil {
			t.Fatalf("%s: unmarshal null: %v", name, err)
		}
		if !nv.IsSet() {
			t.Fatalf("%s: null did not mark the slot present", name)
		}
		if nv.Get() != nil {
			t.Fatalf("%s: null produced a value", name)
		}

		nv.Unset()
		if nv.IsSet() || nv.Get() != nil {
			t.Fatalf("%s: still present after Unset", name)
		}
	})
}

func TestNullableRoundTrip(t *testing.T) {
	roundTripNullable[bool, NullableBool](t, "NullableBool", true, "true")
	roundTripNullable[int, NullableInt](t, "NullableInt", -7, "-7")
	roundTripNullable[int32, NullableInt32](t, "NullableInt32", 2147483647, "2147483647")
	roundTripNullable[int64, NullableInt64](t, "NullableInt64", 9007199254740993, "9007199254740993")
	roundTripNullable[float32, NullableFloat32](t, "NullableFloat32", 1.5, "1.5")
	roundTripNullable[float64, NullableFloat64](t, "NullableFloat64", -0.25, "-0.25")
	roundTripNullable[string, NullableString](t, "NullableString", "a\"b", `"a\"b"`)
	roundTripNullable[time.Time, NullableTime](
		t,
		"NullableTime",
		time.Date(2024, 3, 5, 6, 7, 8, 0, time.UTC),
		`"2024-03-05T06:07:08Z"`,
	)
	roundTripNullable[Date, NullableDate](
		t,
		"NullableDate",
		Date(time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)),
		`"2024-03-05"`,
	)
}

func TestNewNullableConstructors(t *testing.T) {
	b := true
	i := 1
	i32 := int32(2)
	i64 := int64(3)
	f32 := float32(4.5)
	f64 := 5.5
	s := "six"
	tm := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	d := Date(tm)

	cases := []struct {
		name  string
		isSet bool
		got   any
		want  any
	}{
		{"NewNullableBool", NewNullableBool(&b).IsSet(), *NewNullableBool(&b).Get(), b},
		{"NewNullableInt", NewNullableInt(&i).IsSet(), *NewNullableInt(&i).Get(), i},
		{"NewNullableInt32", NewNullableInt32(&i32).IsSet(), *NewNullableInt32(&i32).Get(), i32},
		{"NewNullableInt64", NewNullableInt64(&i64).IsSet(), *NewNullableInt64(&i64).Get(), i64},
		{"NewNullableFloat32", NewNullableFloat32(&f32).IsSet(), *NewNullableFloat32(&f32).Get(), f32},
		{"NewNullableFloat64", NewNullableFloat64(&f64).IsSet(), *NewNullableFloat64(&f64).Get(), f64},
		{"NewNullableString", NewNullableString(&s).IsSet(), *NewNullableString(&s).Get(), s},
		{"NewNullableTime", NewNullableTime(&tm).IsSet(), *NewNullableTime(&tm).Get(), tm},
		{"NewNullableDate", NewNullableDate(&d).IsSet(), *NewNullableDate(&d).Get(), d},
	}
	for _, c := range cases {
		if !c.isSet {
			t.Errorf("%s: constructed wrapper is not set", c.name)
		}
		if c.got != c.want {
			t.Errorf("%s: Get = %v, want %v", c.name, c.got, c.want)
		}
	}

	// A nil argument is the explicit "present and null" the wrapper exists for.
	if v := NewNullableString(nil); !v.IsSet() || v.Get() != nil {
		t.Errorf("NewNullableString(nil): isSet=%v value=%v, want set with no value", v.IsSet(), v.Get())
	}
}

func TestNullableOfLifts(t *testing.T) {
	s := NullableStringOf("x")
	if !s.IsSet() || s.Get() == nil || *s.Get() != "x" {
		t.Errorf("NullableStringOf: %+v", s)
	}
	i := NullableInt64Of(42)
	if !i.IsSet() || i.Get() == nil || *i.Get() != 42 {
		t.Errorf("NullableInt64Of: %+v", i)
	}
	f := NullableFloat64Of(2.5)
	if !f.IsSet() || f.Get() == nil || *f.Get() != 2.5 {
		t.Errorf("NullableFloat64Of: %+v", f)
	}
	b := NullableBoolOf(false)
	if !b.IsSet() || b.Get() == nil || *b.Get() != false {
		t.Errorf("NullableBoolOf: %+v", b)
	}
}

// A lift returns a value, not a pointer, so each call must own its storage —
// otherwise two fields lifted from the same helper alias one slot.
func TestNullableOfDoesNotAliasStorage(t *testing.T) {
	first := NullableStringOf("first")
	second := NullableStringOf("second")
	if *first.Get() != "first" || *second.Get() != "second" {
		t.Fatalf("lifts alias storage: %q / %q", *first.Get(), *second.Get())
	}
}

// The wrapper is a model field before it is anything else, so the shape it
// produces inside a struct is the contract that matters.
func TestNullableInStruct(t *testing.T) {
	type model struct {
		Name  NullableString `json:"name"`
		Count NullableInt64  `json:"count"`
		Day   NullableDate   `json:"day"`
	}
	m := model{
		Name:  NullableStringOf("kit"),
		Count: NullableInt64Of(3),
		Day:   *NewNullableDate(nil),
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	const want = `{"name":"kit","count":3,"day":null}`
	if string(data) != want {
		t.Fatalf("marshal = %s, want %s", data, want)
	}

	var back model
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if *back.Name.Get() != "kit" || *back.Count.Get() != 3 {
		t.Fatalf("round trip lost values: %+v", back)
	}
	if !back.Day.IsSet() || back.Day.Get() != nil {
		t.Fatalf("null day: isSet=%v value=%v", back.Day.IsSet(), back.Day.Get())
	}
}
