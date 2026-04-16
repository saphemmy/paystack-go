package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func TestTransfer_Initiate_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transfer_initiate.json", Status: http.StatusOK})
	tr, err := c.Transfer().Initiate(context.Background(), &paystack.TransferInitiateParams{Source: "balance", Amount: 3000, Recipient: "RCP_x"})
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	if tr.TransferCode == "" {
		t.Fatalf("empty: %+v", tr)
	}
}

func TestTransfer_Verify_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transfer_fetch.json", Status: http.StatusOK})
	tr, err := c.Transfer().Verify(context.Background(), "ref_001")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if tr.Status != "success" {
		t.Fatalf("Status = %q", tr.Status)
	}
}

func TestTransfer_Fetch_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_, _ = c.Transfer().Fetch(context.Background(), "TRF_x")
	if mb.LastCall().Path != "/transfer/TRF_x" {
		t.Fatalf("path = %q", mb.LastCall().Path)
	}
}

func TestTransfer_Finalize_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transfer_fetch.json", Status: http.StatusOK})
	_, err := c.Transfer().Finalize(context.Background(), &paystack.TransferFinalizeParams{TransferCode: "TRF_x", OTP: "123456"})
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
}

func TestTransfer_ErrorMatrix(t *testing.T) {
	methods := map[string]func(paystack.ClientInterface) error{
		"Initiate": func(c paystack.ClientInterface) error {
			_, e := c.Transfer().Initiate(context.Background(), &paystack.TransferInitiateParams{})
			return e
		},
		"Finalize": func(c paystack.ClientInterface) error {
			_, e := c.Transfer().Finalize(context.Background(), &paystack.TransferFinalizeParams{})
			return e
		},
		"Fetch": func(c paystack.ClientInterface) error {
			_, e := c.Transfer().Fetch(context.Background(), "x")
			return e
		},
		"Verify": func(c paystack.ClientInterface) error {
			_, e := c.Transfer().Verify(context.Background(), "x")
			return e
		},
		"List": func(c paystack.ClientInterface) error {
			_, _, e := c.Transfer().List(context.Background(), nil)
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
