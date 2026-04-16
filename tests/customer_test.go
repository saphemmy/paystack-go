package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func TestCustomer_Create_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "customer_create.json", Status: http.StatusOK})
	cust, err := c.Customer().Create(context.Background(), &paystack.CustomerCreateParams{Email: "x@y.z"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cust.CustomerCode == "" {
		t.Fatalf("CustomerCode empty: %+v", cust)
	}
}

func TestCustomer_Create_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, err := c.Customer().Create(context.Background(), &paystack.CustomerCreateParams{Email: "x@y.z"})
		return err
	})
}

func TestCustomer_Fetch_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "customer_fetch.json", Status: http.StatusOK})
	cust, err := c.Customer().Fetch(context.Background(), "CUS_abc")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if cust.TotalVolume != 150000 {
		t.Fatalf("TotalVolume = %d", cust.TotalVolume)
	}
}

func TestCustomer_Fetch_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, err := c.Customer().Fetch(context.Background(), "CUS_x")
		return err
	})
}

func TestCustomer_List_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "customer_list.json", Status: http.StatusOK})
	custs, meta, err := c.Customer().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(custs) == 0 {
		t.Fatal("empty")
	}
	if meta.Total != 1 {
		t.Fatalf("meta.Total = %d", meta.Total)
	}
}

func TestCustomer_Update_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	firstName := "A"
	_, _ = c.Customer().Update(context.Background(), "CUS_x", &paystack.CustomerUpdateParams{FirstName: &firstName})
	last := mb.LastCall()
	if last.Method != http.MethodPut || last.Path != "/customer/CUS_x" {
		t.Fatalf("call shape wrong: %+v", last)
	}
}

func TestCustomer_SetRiskAction_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "customer_fetch.json", Status: http.StatusOK})
	_, err := c.Customer().SetRiskAction(context.Background(), &paystack.CustomerRiskActionParams{
		Customer: "CUS_x", RiskAction: "allow",
	})
	if err != nil {
		t.Fatalf("SetRiskAction: %v", err)
	}
}

func TestCustomer_DeactivateAuthorization_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	err := c.Customer().DeactivateAuthorization(context.Background(), "AUTH_x")
	if err != nil {
		t.Fatalf("DeactivateAuthorization: %v", err)
	}
	last := mb.LastCall()
	if last.Path != "/customer/deactivate_authorization" {
		t.Fatalf("path = %q", last.Path)
	}
}

func TestCustomer_DeactivateAuthorization_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		return c.Customer().DeactivateAuthorization(context.Background(), "AUTH_x")
	})
}

func TestCustomer_List_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, _, err := c.Customer().List(context.Background(), nil)
		return err
	})
}

func TestCustomer_Update_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, err := c.Customer().Update(context.Background(), "CUS_x", &paystack.CustomerUpdateParams{})
		return err
	})
}

func TestCustomer_SetRiskAction_ErrorMatrix(t *testing.T) {
	testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
		c := newTestClient(t, b)
		_, err := c.Customer().SetRiskAction(context.Background(), &paystack.CustomerRiskActionParams{Customer: "CUS_x", RiskAction: "allow"})
		return err
	})
}
