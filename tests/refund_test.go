package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func TestRefund_Create_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "refund_create.json", Status: http.StatusOK})
	r, err := c.Refund().Create(context.Background(), &paystack.RefundCreateParams{Transaction: "4099260516"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == 0 {
		t.Fatalf("empty: %+v", r)
	}
}

func TestRefund_Fetch_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "refund_fetch.json", Status: http.StatusOK})
	r, err := c.Refund().Fetch(context.Background(), "3018284")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if r.Status != "processed" {
		t.Fatalf("Status = %q", r.Status)
	}
}

func TestRefund_List_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_, _, _ = c.Refund().List(context.Background(), nil)
	if mb.LastCall().Path != "/refund" {
		t.Fatalf("path = %q", mb.LastCall().Path)
	}
}

func TestRefund_ErrorMatrix(t *testing.T) {
	methods := map[string]func(paystack.ClientInterface) error{
		"Create": func(c paystack.ClientInterface) error {
			_, e := c.Refund().Create(context.Background(), &paystack.RefundCreateParams{})
			return e
		},
		"Fetch": func(c paystack.ClientInterface) error { _, e := c.Refund().Fetch(context.Background(), "x"); return e },
		"List": func(c paystack.ClientInterface) error {
			_, _, e := c.Refund().List(context.Background(), nil)
			return e
		},
	}
	for name, m := range methods {
		name, m := name, m
		t.Run(name, func(t *testing.T) {
			testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
				return m(newTestClient(t, b))
			})
		})
	}
}
