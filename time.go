package paystack

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Time wraps time.Time so the SDK can accept every date layout the Paystack
// API emits. Layouts are tried in order; the first one that parses wins.
// An empty JSON string, literal "null", or a JSON null yields the zero Time.
type Time struct {
	time.Time
}

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.000-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// UnmarshalJSON parses a timestamp from any layout Paystack emits.
func (t *Time) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	for _, layout := range timeLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("paystack: cannot parse %q as time", s)
}

// MarshalJSON emits an RFC 3339 timestamp, or JSON null for the zero value.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}
