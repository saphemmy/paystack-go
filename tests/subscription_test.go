package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func TestSubscription_Create_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "subscription_create.json", Status: http.StatusOK})
	sub, err := c.Subscription().Create(context.Background(), &paystack.SubscriptionCreateParams{Customer: "CUS_x", Plan: "PLN_x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sub.SubscriptionCode == "" {
		t.Fatalf("empty sub: %+v", sub)
	}
}

func TestSubscription_Fetch_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "subscription_fetch.json", Status: http.StatusOK})
	sub, err := c.Subscription().Fetch(context.Background(), "SUB_x")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if sub.Status != "active" {
		t.Fatalf("Status = %q", sub.Status)
	}
}

func TestSubscription_EnableDisable_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_ = c.Subscription().Enable(context.Background(), &paystack.SubscriptionToggleParams{Code: "SUB_x", Token: "t"})
	if mb.LastCall().Path != "/subscription/enable" {
		t.Fatalf("enable path = %q", mb.LastCall().Path)
	}
	_ = c.Subscription().Disable(context.Background(), &paystack.SubscriptionToggleParams{Code: "SUB_x", Token: "t"})
	if mb.LastCall().Path != "/subscription/disable" {
		t.Fatalf("disable path = %q", mb.LastCall().Path)
	}
}

func TestSubscription_GenerateUpdateLink_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_, _ = c.Subscription().GenerateUpdateLink(context.Background(), "SUB_x")
	if mb.LastCall().Path != "/subscription/SUB_x/manage/link" {
		t.Fatalf("path = %q", mb.LastCall().Path)
	}
}

func TestSubscription_ErrorMatrix(t *testing.T) {
	methods := map[string]func(paystack.ClientInterface) error{
		"Create": func(c paystack.ClientInterface) error {
			_, e := c.Subscription().Create(context.Background(), &paystack.SubscriptionCreateParams{})
			return e
		},
		"Fetch": func(c paystack.ClientInterface) error {
			_, e := c.Subscription().Fetch(context.Background(), "x")
			return e
		},
		"List": func(c paystack.ClientInterface) error {
			_, _, e := c.Subscription().List(context.Background(), nil)
			return e
		},
		"Enable": func(c paystack.ClientInterface) error {
			return c.Subscription().Enable(context.Background(), &paystack.SubscriptionToggleParams{})
		},
		"Disable": func(c paystack.ClientInterface) error {
			return c.Subscription().Disable(context.Background(), &paystack.SubscriptionToggleParams{})
		},
		"GenerateUpdateLink": func(c paystack.ClientInterface) error {
			_, e := c.Subscription().GenerateUpdateLink(context.Background(), "x")
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
