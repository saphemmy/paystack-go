package paystack_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func newTestClient(t *testing.T, b paystack.Backend) paystack.ClientInterface {
	t.Helper()
	c, err := paystack.New("sk_test_xxx", paystack.WithBackend(b))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestTransaction_Initialize_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_initialize.json", Status: http.StatusOK})
	data, err := c.Transaction().Initialize(context.Background(), &paystack.TransactionInitializeParams{
		Email: "x@y.z", Amount: 5000,
	})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if data.AuthorizationURL == "" || data.Reference == "" || data.AccessCode == "" {
		t.Fatalf("Initialize returned empty data: %+v", data)
	}
}

func TestTransaction_Initialize_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, err := c.Transaction().Initialize(context.Background(), &paystack.TransactionInitializeParams{Email: "x@y", Amount: 1})
		return err
	})
}

func TestTransaction_Verify_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_verify.json", Status: http.StatusOK})
	tx, err := c.Transaction().Verify(context.Background(), "ref")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if tx.Reference != "re4lyvq3s3" || tx.Status != "success" {
		t.Fatalf("Verify returned %+v", tx)
	}
	if tx.Customer == nil || tx.Customer.Email != "test@example.com" {
		t.Fatalf("Customer not populated: %+v", tx.Customer)
	}
	if tx.Authorization == nil || tx.Authorization.AuthorizationCode == "" {
		t.Fatalf("Authorization not populated: %+v", tx.Authorization)
	}
}

func TestTransaction_Verify_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, err := c.Transaction().Verify(context.Background(), "ref")
		return err
	})
}

func TestTransaction_List_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_list.json", Status: http.StatusOK})
	txs, meta, err := c.Transaction().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("len(txs) = %d, want 2", len(txs))
	}
	if meta.Total != 2 {
		t.Fatalf("meta.Total = %d, want 2", meta.Total)
	}
}

func TestTransaction_List_EmptyIsNotAnError(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_list_empty.json", Status: http.StatusOK})
	txs, meta, err := c.Transaction().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("empty list should not error, got %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("expected empty slice, got %+v", txs)
	}
	if meta.Total != 120 {
		t.Fatalf("meta.Total = %d, want 120", meta.Total)
	}
}

func TestTransaction_List_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, _, err := c.Transaction().List(context.Background(), nil)
		return err
	})
}

func TestTransaction_Fetch_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_verify.json", Status: http.StatusOK})
	tx, err := c.Transaction().Fetch(context.Background(), 4099260516)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if tx.ID != 4099260516 {
		t.Fatalf("ID = %d", tx.ID)
	}
}

func TestTransaction_Fetch_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_, _ = c.Transaction().Fetch(context.Background(), 42)
	last := mb.LastCall()
	if last.Method != http.MethodGet || last.Path != "/transaction/42" {
		t.Fatalf("call shape wrong: %+v", last)
	}
}

func TestTransaction_ChargeAuthorization_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_verify.json", Status: http.StatusOK})
	tx, err := c.Transaction().ChargeAuthorization(context.Background(), &paystack.TransactionChargeAuthorizationParams{
		Email:             "x@y.z",
		Amount:            100,
		AuthorizationCode: "AUTH_xxx",
	})
	if err != nil {
		t.Fatalf("ChargeAuthorization: %v", err)
	}
	if tx.Reference == "" {
		t.Fatal("empty reference")
	}
}

func TestTransaction_Totals_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_, _ = c.Transaction().Totals(context.Background(), nil)
	last := mb.LastCall()
	if last.Path != "/transaction/totals" {
		t.Fatalf("path = %q", last.Path)
	}
}

func TestTransaction_Initialize_PropagatesMockErr(t *testing.T) {
	wantErr := errors.New("boom")
	mb := &testutil.MockBackend{Err: wantErr}
	c := newTestClient(t, mb)
	_, err := c.Transaction().Initialize(context.Background(), &paystack.TransactionInitializeParams{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("want %v, got %v", wantErr, err)
	}
}

func TestTransaction_RemainingErrorMatrices(t *testing.T) {
	methods := map[string]func(paystack.ClientInterface) error{
		"Fetch": func(c paystack.ClientInterface) error {
			_, e := c.Transaction().Fetch(context.Background(), 1)
			return e
		},
		"ChargeAuthorization": func(c paystack.ClientInterface) error {
			_, e := c.Transaction().ChargeAuthorization(context.Background(), &paystack.TransactionChargeAuthorizationParams{})
			return e
		},
		"Totals": func(c paystack.ClientInterface) error {
			_, e := c.Transaction().Totals(context.Background(), nil)
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

func TestTransaction_Concurrent(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_verify.json", Status: http.StatusOK})
	testutil.RunConcurrent(t, 50, func() error {
		_, err := c.Transaction().Verify(context.Background(), "ref")
		return err
	})
}
