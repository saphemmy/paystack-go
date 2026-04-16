package paystack

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "simple error without fields",
			err:  &Error{Code: ErrCodeInvalidKey, Message: "Invalid API key"},
			want: "paystack: Invalid API key (invalid_key)",
		},
		{
			name: "validation error with fields",
			err: &Error{
				Code:    ErrCodeInvalidRequest,
				Message: "Validation failed",
				Fields:  map[string]string{"email": "is required"},
			},
			want: "paystack: Validation failed (invalid_request) — email: is required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestError_ErrorsAs(t *testing.T) {
	var err error = &Error{Code: ErrCodeNotFound, Message: "missing"}
	var target *Error
	if !errors.As(err, &target) {
		t.Fatal("errors.As did not match *Error")
	}
	if target.Code != ErrCodeNotFound {
		t.Fatalf("Code = %q, want %q", target.Code, ErrCodeNotFound)
	}
}

func TestCodeForStatus(t *testing.T) {
	tests := []struct {
		status int
		want   ErrorCode
	}{
		{http.StatusUnauthorized, ErrCodeInvalidKey},
		{http.StatusForbidden, ErrCodeInvalidKey},
		{http.StatusNotFound, ErrCodeNotFound},
		{http.StatusTooManyRequests, ErrCodeRateLimited},
		{http.StatusBadRequest, ErrCodeInvalidRequest},
		{http.StatusUnprocessableEntity, ErrCodeInvalidRequest},
		{http.StatusInternalServerError, ErrCodeServerError},
		{http.StatusBadGateway, ErrCodeServerError},
		{http.StatusServiceUnavailable, ErrCodeServerError},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			if got := codeForStatus(tc.status); got != tc.want {
				t.Fatalf("codeForStatus(%d) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestParseError_JSONEnvelope(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	body := []byte(`{"status":false,"message":"Validation failed","errors":{"email":"is required","amount":"must be positive"}}`)

	err := ParseError(resp, body)
	if err.Code != ErrCodeInvalidRequest {
		t.Fatalf("Code = %q, want %q", err.Code, ErrCodeInvalidRequest)
	}
	if err.Message != "Validation failed" {
		t.Fatalf("Message = %q, want %q", err.Message, "Validation failed")
	}
	if len(err.Fields) != 2 {
		t.Fatalf("Fields length = %d, want 2", len(err.Fields))
	}
	if err.Fields["email"] != "is required" {
		t.Fatalf("Fields[email] = %q, want %q", err.Fields["email"], "is required")
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d, want %d", err.StatusCode, http.StatusBadRequest)
	}
}

func TestParseError_HTMLBody(t *testing.T) {
	tests := []struct {
		name   string
		header string
		body   []byte
	}{
		{"via content-type", "text/html; charset=utf-8", []byte(`<html><body>502 Bad Gateway</body></html>`)},
		{"via sniff doctype", "", []byte(`<!doctype html><html>nope</html>`)},
		{"via sniff html", "", []byte(`<html><body>oops</body></html>`)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     http.Header{"Content-Type": []string{tc.header}},
			}
			err := ParseError(resp, tc.body)
			if err.Code != ErrCodeServerError {
				t.Fatalf("Code = %q, want %q", err.Code, ErrCodeServerError)
			}
			if err.Message == "" {
				t.Fatal("Message should not be empty")
			}
		})
	}
}

func TestParseError_RetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"seconds", "30", 30 * time.Second},
		{"zero seconds", "0", 0},
		{"empty", "", 0},
		{"garbage", "nope", 0},
		{"past http date", "Mon, 02 Jan 2006 15:04:05 GMT", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": []string{tc.header}},
			}
			err := ParseError(resp, []byte(`{"status":false,"message":"rate limited"}`))
			if err.Code != ErrCodeRateLimited {
				t.Fatalf("Code = %q, want %q", err.Code, ErrCodeRateLimited)
			}
			if err.RetryAfter != tc.want {
				t.Fatalf("RetryAfter = %v, want %v", err.RetryAfter, tc.want)
			}
		})
	}
}

func TestParseError_FallbackOnBadJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	err := ParseError(resp, []byte(`not json at all`))
	if err.Code != ErrCodeServerError {
		t.Fatalf("Code = %q, want %q", err.Code, ErrCodeServerError)
	}
	if err.Message == "" {
		t.Fatal("Message should not be empty")
	}
}

func TestParseError_UnknownStatusFallback(t *testing.T) {
	resp := &http.Response{
		StatusCode: 599,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	err := ParseError(resp, []byte(`garbage`))
	if err.Code != ErrCodeServerError {
		t.Fatalf("Code = %q, want %q", err.Code, ErrCodeServerError)
	}
	if err.Message == "" {
		t.Fatal("Message should fall back to a non-empty string")
	}
}

func TestParseRetryAfter_FutureHTTPDate(t *testing.T) {
	future := time.Now().UTC().Add(42 * time.Second).Format(http.TimeFormat)
	got := parseRetryAfter(future)
	if got < 30*time.Second || got > 60*time.Second {
		t.Fatalf("RetryAfter = %v, want ~42s", got)
	}
}

func TestDecodeFields(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{"empty", ``, nil},
		{"null", `null`, nil},
		{"string map", `{"email":"bad"}`, map[string]string{"email": "bad"}},
		{"non-string values coerced", `{"amount":42,"email":"bad"}`, map[string]string{"amount": "42", "email": "bad"}},
		{"empty object", `{}`, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decodeFields([]byte(tc.in))
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tc.want))
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Fatalf("Fields[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
