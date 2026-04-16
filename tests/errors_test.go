package paystack_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

// All error-code classification, HTML sniffing, Retry-After parsing, and
// validation-fields extraction is covered through the exported ParseError
// entry point. The internal helpers codeForStatus / isHTML / parseRetryAfter
// / decodeFields are reached by every row in the table below.

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *paystack.Error
		want string
	}{
		{
			name: "simple error without fields",
			err:  &paystack.Error{Code: paystack.ErrCodeInvalidKey, Message: "Invalid API key"},
			want: "paystack: Invalid API key (invalid_key)",
		},
		{
			name: "validation error with fields",
			err: &paystack.Error{
				Code:    paystack.ErrCodeInvalidRequest,
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
	var err error = &paystack.Error{Code: paystack.ErrCodeNotFound, Message: "missing"}
	var target *paystack.Error
	if !errors.As(err, &target) {
		t.Fatal("errors.As did not match *Error")
	}
	if target.Code != paystack.ErrCodeNotFound {
		t.Fatalf("Code = %q", target.Code)
	}
}

func TestParseError_StatusClassification(t *testing.T) {
	tests := []struct {
		status int
		want   paystack.ErrorCode
	}{
		{http.StatusUnauthorized, paystack.ErrCodeInvalidKey},
		{http.StatusForbidden, paystack.ErrCodeInvalidKey},
		{http.StatusNotFound, paystack.ErrCodeNotFound},
		{http.StatusTooManyRequests, paystack.ErrCodeRateLimited},
		{http.StatusBadRequest, paystack.ErrCodeInvalidRequest},
		{http.StatusUnprocessableEntity, paystack.ErrCodeInvalidRequest},
		{http.StatusInternalServerError, paystack.ErrCodeServerError},
		{http.StatusBadGateway, paystack.ErrCodeServerError},
		{http.StatusServiceUnavailable, paystack.ErrCodeServerError},
		{599, paystack.ErrCodeServerError},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.status,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}
			err := paystack.ParseError(resp, []byte(`{"status":false,"message":"m"}`))
			if err.Code != tc.want {
				t.Fatalf("Code = %q, want %q", err.Code, tc.want)
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
	err := paystack.ParseError(resp, body)
	if err.Code != paystack.ErrCodeInvalidRequest {
		t.Fatalf("Code = %q", err.Code)
	}
	if err.Message != "Validation failed" {
		t.Fatalf("Message = %q", err.Message)
	}
	if len(err.Fields) != 2 || err.Fields["email"] != "is required" {
		t.Fatalf("Fields = %+v", err.Fields)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d", err.StatusCode)
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
			err := paystack.ParseError(resp, tc.body)
			if err.Code != paystack.ErrCodeServerError {
				t.Fatalf("Code = %q", err.Code)
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
			err := paystack.ParseError(resp, []byte(`{"status":false,"message":"rate limited"}`))
			if err.Code != paystack.ErrCodeRateLimited {
				t.Fatalf("Code = %q", err.Code)
			}
			if err.RetryAfter != tc.want {
				t.Fatalf("RetryAfter = %v, want %v", err.RetryAfter, tc.want)
			}
		})
	}
}

func TestParseError_FutureHTTPDate(t *testing.T) {
	future := time.Now().UTC().Add(42 * time.Second).Format(http.TimeFormat)
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{future}, "Content-Type": []string{"application/json"}},
	}
	err := paystack.ParseError(resp, []byte(`{"status":false,"message":"too many"}`))
	if err.RetryAfter < 30*time.Second || err.RetryAfter > 60*time.Second {
		t.Fatalf("RetryAfter = %v, want ~42s", err.RetryAfter)
	}
}

func TestParseError_FallbackOnBadJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	err := paystack.ParseError(resp, []byte(`not json at all`))
	if err.Code != paystack.ErrCodeServerError {
		t.Fatalf("Code = %q", err.Code)
	}
	if err.Message == "" {
		t.Fatal("Message should not be empty")
	}
}

func TestParseError_UnknownStatusMessage(t *testing.T) {
	resp := &http.Response{
		StatusCode: 599,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	err := paystack.ParseError(resp, []byte(`garbage`))
	if err.Message == "" {
		t.Fatal("Message must not be empty for unknown status")
	}
}

func TestParseError_FieldsWithNonStringValues(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	body := []byte(`{"status":false,"message":"bad","errors":{"amount":42,"email":"bad"}}`)
	err := paystack.ParseError(resp, body)
	if err.Fields["amount"] != "42" || err.Fields["email"] != "bad" {
		t.Fatalf("Fields = %+v", err.Fields)
	}
}
