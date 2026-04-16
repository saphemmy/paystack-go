package testutil

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

// ErrorCase captures one row of the standard error matrix every service method
// must pass. Use AssertErrorMatrix to run the whole table in a single test.
type ErrorCase struct {
	Name       string
	Fixture    string
	Status     int
	Header     http.Header
	WantCode   paystack.ErrorCode
	WantRetry  time.Duration
	WantFields bool
}

// DefaultErrorMatrix returns the seven error rows every service method is
// required to handle by the spec.
func DefaultErrorMatrix() []ErrorCase {
	return []ErrorCase{
		{Name: "401", Fixture: "error_401.json", Status: http.StatusUnauthorized, WantCode: paystack.ErrCodeInvalidKey},
		{Name: "400 with fields", Fixture: "error_400_fields.json", Status: http.StatusBadRequest, WantCode: paystack.ErrCodeInvalidRequest, WantFields: true},
		{Name: "404", Fixture: "error_404.json", Status: http.StatusNotFound, WantCode: paystack.ErrCodeNotFound},
		{Name: "429", Fixture: "error_429.json", Status: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"9"}}, WantCode: paystack.ErrCodeRateLimited, WantRetry: 9 * time.Second},
		{Name: "500", Fixture: "error_500.json", Status: http.StatusInternalServerError, WantCode: paystack.ErrCodeServerError},
		{Name: "HTML body", Fixture: "error_html.html", Status: http.StatusBadGateway, WantCode: paystack.ErrCodeServerError},
	}
}

// AssertErrorMatrix runs DefaultErrorMatrix against invoke. invoke is called
// once per row with a FixtureBackend wired to that row's fixture and must
// return the service method's error.
func AssertErrorMatrix(t *testing.T, invoke func(paystack.Backend) error) {
	t.Helper()
	for _, tc := range DefaultErrorMatrix() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			fb := &FixtureBackend{Fixture: tc.Fixture, Status: tc.Status, Header: tc.Header}
			err := invoke(fb)
			var pErr *paystack.Error
			if !errors.As(err, &pErr) {
				t.Fatalf("expected *paystack.Error, got %T: %v", err, err)
			}
			if pErr.Code != tc.WantCode {
				t.Fatalf("Code = %q, want %q", pErr.Code, tc.WantCode)
			}
			if tc.WantFields && len(pErr.Fields) == 0 {
				t.Fatal("expected Fields to be populated")
			}
			if tc.WantRetry != 0 && pErr.RetryAfter != tc.WantRetry {
				t.Fatalf("RetryAfter = %v, want %v", pErr.RetryAfter, tc.WantRetry)
			}
		})
	}
}

// RunConcurrent invokes fn n times in parallel and fails the test if any
// call returns an error. Ensures service methods are safe for concurrent use.
func RunConcurrent(t *testing.T, n int, fn func() error) {
	t.Helper()
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent call failed: %v", err)
	}
}
