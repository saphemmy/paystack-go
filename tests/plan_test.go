package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func TestPlan_Create_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "plan_create.json", Status: http.StatusOK})
	p, err := c.Plan().Create(context.Background(), &paystack.PlanCreateParams{Name: "Monthly Gold", Amount: 500000, Interval: "monthly"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.PlanCode == "" {
		t.Fatalf("empty plan: %+v", p)
	}
}

func TestPlan_Fetch_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "plan_fetch.json", Status: http.StatusOK})
	p, err := c.Plan().Fetch(context.Background(), "PLN_x")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if p.Amount != 500000 {
		t.Fatalf("Amount = %d", p.Amount)
	}
}

func TestPlan_List_EmptyIsOK(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "transaction_list_empty.json", Status: http.StatusOK})
	plans, meta, err := c.Plan().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plans) != 0 || meta.Total != 120 {
		t.Fatalf("plans=%d meta=%+v", len(plans), meta)
	}
}

func TestPlan_Update_NilResponseStillOK(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	if err := c.Plan().Update(context.Background(), "PLN_x", &paystack.PlanUpdateParams{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	last := mb.LastCall()
	if last.Method != http.MethodPut || last.Path != "/plan/PLN_x" {
		t.Fatalf("call shape wrong: %+v", last)
	}
}

func TestPlan_ErrorMatrix(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
			_, err := newTestClient(t, b).Plan().Create(context.Background(), &paystack.PlanCreateParams{})
			return err
		})
	})
	t.Run("Fetch", func(t *testing.T) {
		testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
			_, err := newTestClient(t, b).Plan().Fetch(context.Background(), "x")
			return err
		})
	})
	t.Run("List", func(t *testing.T) {
		testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
			_, _, err := newTestClient(t, b).Plan().List(context.Background(), nil)
			return err
		})
	})
	t.Run("Update", func(t *testing.T) {
		testutil.AssertErrorMatrix(t, func(b paystack.Backend) error {
			return newTestClient(t, b).Plan().Update(context.Background(), "x", &paystack.PlanUpdateParams{})
		})
	})
}
