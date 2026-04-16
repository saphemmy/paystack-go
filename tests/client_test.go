package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
)

// nopBackend satisfies Backend without doing anything. Used to verify Client
// wiring without touching HTTP.
type nopBackend struct{}

func (nopBackend) Call(context.Context, string, string, interface{}, interface{}) error {
	return nil
}
func (nopBackend) CallRaw(context.Context, string, string, interface{}) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func TestClient_ServicesReturnSameInstance(t *testing.T) {
	c, err := paystack.New("sk_test_x", paystack.WithBackend(&nopBackend{}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Transaction() == nil {
		t.Fatal("Transaction is nil")
	}
	if c.Customer() == nil {
		t.Fatal("Customer is nil")
	}
	if c.Plan() == nil {
		t.Fatal("Plan is nil")
	}
	if c.Subscription() == nil {
		t.Fatal("Subscription is nil")
	}
	if c.Transfer() == nil {
		t.Fatal("Transfer is nil")
	}
	if c.Charge() == nil {
		t.Fatal("Charge is nil")
	}
	if c.Refund() == nil {
		t.Fatal("Refund is nil")
	}
	if c.Transaction() != c.Transaction() {
		t.Fatal("Transaction returned different instances")
	}
}

func TestClient_BackendAccessor(t *testing.T) {
	b := &nopBackend{}
	c, _ := paystack.New("sk_test_x", paystack.WithBackend(b))
	if c.Backend() != b {
		t.Fatal("Backend() did not return wired backend")
	}
}
