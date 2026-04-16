// Package paystack provides a Go client for the Paystack API.
//
// All monetary amounts are in kobo (1 NGN = 100 kobo) and are never silently
// converted. Callers are responsible for currency handling.
package paystack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrorCode classifies errors returned by the Paystack API.
type ErrorCode string

const (
	ErrCodeInvalidKey     ErrorCode = "invalid_key"
	ErrCodeInvalidRequest ErrorCode = "invalid_request"
	ErrCodeNotFound       ErrorCode = "not_found"
	ErrCodeRateLimited    ErrorCode = "rate_limited"
	ErrCodeServerError    ErrorCode = "server_error"
)

// Error is the single error type returned for every non-2xx response from the
// Paystack API. Callers switch on Code to branch behaviour. Works with
// errors.As.
type Error struct {
	Code       ErrorCode
	Message    string
	StatusCode int
	Fields     map[string]string
	RetryAfter time.Duration
	RawBody    []byte
}

func (e *Error) Error() string {
	if len(e.Fields) > 0 {
		parts := make([]string, 0, len(e.Fields))
		for k, v := range e.Fields {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v))
		}
		return fmt.Sprintf("paystack: %s (%s) — %s", e.Message, e.Code, strings.Join(parts, "; "))
	}
	return fmt.Sprintf("paystack: %s (%s)", e.Message, e.Code)
}

type errorEnvelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Errors  json.RawMessage `json:"errors,omitempty"`
}

// parseError converts a non-2xx HTTP response into a *Error. body is the
// already-read response body; caller is responsible for draining the response.
func parseError(resp *http.Response, body []byte) *Error {
	code := codeForStatus(resp.StatusCode)
	err := &Error{
		Code:       code,
		StatusCode: resp.StatusCode,
		RawBody:    body,
	}

	if isHTML(resp.Header.Get("Content-Type"), body) {
		err.Code = ErrCodeServerError
		err.Message = fmt.Sprintf("unexpected HTML response from Paystack (status %d)", resp.StatusCode)
		return err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		err.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	}

	var env errorEnvelope
	if jsonErr := json.Unmarshal(body, &env); jsonErr == nil && env.Message != "" {
		err.Message = env.Message
		err.Fields = decodeFields(env.Errors)
	} else {
		err.Message = http.StatusText(resp.StatusCode)
		if err.Message == "" {
			err.Message = fmt.Sprintf("unexpected status %d", resp.StatusCode)
		}
	}

	return err
}

func codeForStatus(status int) ErrorCode {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return ErrCodeInvalidKey
	case status == http.StatusNotFound:
		return ErrCodeNotFound
	case status == http.StatusTooManyRequests:
		return ErrCodeRateLimited
	case status >= 400 && status < 500:
		return ErrCodeInvalidRequest
	default:
		return ErrCodeServerError
	}
}

func isHTML(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	trimmed := strings.TrimLeft(string(body), " \t\r\n")
	return strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html") ||
		strings.HasPrefix(strings.ToLower(trimmed), "<html")
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func decodeFields(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var asStringMap map[string]string
	if err := json.Unmarshal(raw, &asStringMap); err == nil && len(asStringMap) > 0 {
		return asStringMap
	}
	var asAnyMap map[string]interface{}
	if err := json.Unmarshal(raw, &asAnyMap); err == nil && len(asAnyMap) > 0 {
		out := make(map[string]string, len(asAnyMap))
		for k, v := range asAnyMap {
			out[k] = fmt.Sprintf("%v", v)
		}
		return out
	}
	return nil
}
