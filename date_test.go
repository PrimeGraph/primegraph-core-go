package primegraphcore

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateMarshalJSON(t *testing.T) {
	cases := []struct {
		name  string
		value time.Time
		want  string
	}{
		{
			name:  "UTC midnight",
			value: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
			want:  `"2024-03-05"`,
		},
		{
			// A day names no time of day, so whatever hour the carrier holds is
			// dropped rather than rounded.
			name:  "time of day is dropped",
			value: time.Date(2024, 3, 5, 23, 59, 59, 0, time.UTC),
			want:  `"2024-03-05"`,
		},
		{
			// The writer normalises to UTC first, so an instant that is still the
			// 5th locally but already the 6th in UTC writes the 6th.
			name:  "offset is normalised to UTC",
			value: time.Date(2024, 3, 5, 23, 0, 0, 0, time.FixedZone("plus5", 5*3600)),
			want:  `"2024-03-05"`,
		},
		{
			name:  "offset crossing into the next UTC day",
			value: time.Date(2024, 3, 5, 23, 0, 0, 0, time.FixedZone("minus5", -5*3600)),
			want:  `"2024-03-06"`,
		},
		{
			name:  "single-digit month and day are padded",
			value: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			want:  `"2024-01-02"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := json.Marshal(Date(c.value))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(data) != c.want {
				t.Fatalf("marshal = %s, want %s", data, c.want)
			}
		})
	}
}

func TestDateUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want time.Time
	}{
		{
			name: "day alone lands on UTC midnight",
			raw:  `"2024-03-05"`,
			want: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			// A store that keeps the field as its own instant type hands the same
			// day back as a full moment; its UTC day part is the value.
			name: "full instant keeps its UTC day",
			raw:  `"2024-03-05T22:10:00Z"`,
			want: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "offset instant is converted before the day is taken",
			raw:  `"2024-03-05T01:00:00+05:00"`,
			want: time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "fractional seconds are accepted",
			raw:  `"2024-03-05T22:10:00.123456Z"`,
			want: time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var d Date
			if err := json.Unmarshal([]byte(c.raw), &d); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got := time.Time(d)
			if !got.Equal(c.want) {
				t.Fatalf("unmarshal = %s, want %s", got, c.want)
			}
			if got.Location() != time.UTC {
				t.Fatalf("location = %s, want UTC", got.Location())
			}
		})
	}
}

func TestDateUnmarshalRejectsNonDates(t *testing.T) {
	for _, raw := range []string{`"not a date"`, `"2024-13-45"`, `12345`, `{}`, `"2024/03/05"`} {
		var d Date
		if err := json.Unmarshal([]byte(raw), &d); err == nil {
			t.Errorf("unmarshal(%s) succeeded, want an error", raw)
		}
	}
}

func TestDateRoundTrip(t *testing.T) {
	original := Date(time.Date(2031, 12, 31, 0, 0, 0, 0, time.UTC))
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Date
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !time.Time(back).Equal(time.Time(original)) {
		t.Fatalf("round trip = %s, want %s", time.Time(back), time.Time(original))
	}
}
