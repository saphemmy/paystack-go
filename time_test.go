package paystack

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
		check   func(t *testing.T, got Time)
	}{
		{
			name: "RFC3339",
			in:   `"2024-01-15T10:30:00Z"`,
			check: func(t *testing.T, got Time) {
				if got.Year() != 2024 || got.Month() != time.January || got.Day() != 15 {
					t.Fatalf("date mismatch: %v", got.Time)
				}
			},
		},
		{
			name: "RFC3339 with milliseconds",
			in:   `"2024-01-15T10:30:00.123Z"`,
			check: func(t *testing.T, got Time) {
				if got.Nanosecond() == 0 {
					t.Fatal("nanoseconds should be preserved")
				}
			},
		},
		{
			name: "space separator",
			in:   `"2024-01-15 10:30:00"`,
			check: func(t *testing.T, got Time) {
				if got.Hour() != 10 || got.Minute() != 30 {
					t.Fatalf("time mismatch: %v", got.Time)
				}
			},
		},
		{
			name: "date only",
			in:   `"2024-01-15"`,
			check: func(t *testing.T, got Time) {
				if got.Year() != 2024 {
					t.Fatalf("year mismatch: %v", got.Time)
				}
			},
		},
		{
			name: "RFC3339 with offset",
			in:   `"2024-01-15T10:30:00+01:00"`,
			check: func(t *testing.T, got Time) {
				if got.Hour() != 10 {
					t.Fatalf("hour mismatch: %v", got.Time)
				}
			},
		},
		{
			name: "empty string",
			in:   `""`,
			check: func(t *testing.T, got Time) {
				if !got.IsZero() {
					t.Fatal("expected zero Time for empty string")
				}
			},
		},
		{
			name: "literal null",
			in:   `null`,
			check: func(t *testing.T, got Time) {
				if !got.IsZero() {
					t.Fatal("expected zero Time for null")
				}
			},
		},
		{
			name:    "garbage",
			in:      `"not a date"`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got Time
			err := got.UnmarshalJSON([]byte(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   Time
		want string
	}{
		{
			name: "zero value is null",
			in:   Time{},
			want: "null",
		},
		{
			name: "non-zero value is RFC3339",
			in:   Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
			want: `"2024-01-15T10:30:00Z"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestTime_RoundTrip(t *testing.T) {
	original := Time{Time: time.Date(2024, 6, 12, 14, 30, 0, 0, time.UTC)}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded Time
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !decoded.Equal(original.Time) {
		t.Fatalf("round trip mismatch: %v != %v", decoded.Time, original.Time)
	}
}
