package primegraphcore

import (
	"encoding/json"
	"time"
)

// Date is the calendar day a `format: date` field stores. It carries the same
// instant a moment does — only the wire form differs, which is why it needs a
// type of its own for encoding/json to find these methods.
type Date time.Time

// MarshalJSON writes the day alone, the form every other language writes.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).UTC().Format("2006-01-02"))
}

// UnmarshalJSON reads the day back onto the instant its UTC midnight names.
// The reader is wider than the writer: a store that keeps the field as its own
// instant type hands the same day back as a full moment, and its day part is
// the value.
func (d *Date) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		*d = Date(parsed)
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return err
	}
	utc := parsed.UTC()
	*d = Date(time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC))
	return nil
}
