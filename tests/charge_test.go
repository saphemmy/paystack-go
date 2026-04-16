package paystack_test

import (
	"context"
	"net/http"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/internal/testutil"
)

func TestCharge_Create_Success(t *testing.T) {
	c := newTestClient(t, &testutil.FixtureBackend{Fixture: "charge_create.json", Status: http.StatusOK})
	res, err := c.Charge().Create(context.Background(), &paystack.ChargeCreateParams{Email: "x@y.z", Amount: 40000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Status != "send_otp" {
		t.Fatalf("Status = %q", res.Status)
	}
}

func TestCharge_SubmitPaths(t *testing.T) {
	paths := map[string]func(paystack.ClientInterface) (string, error){
		"/charge/submit_pin": func(c paystack.ClientInterface) (string, error) {
			_, err := c.Charge().SubmitPin(context.Background(), &paystack.ChargeSubmitPinParams{Reference: "r", PIN: "1234"})
			return "/charge/submit_pin", err
		},
		"/charge/submit_otp": func(c paystack.ClientInterface) (string, error) {
			_, err := c.Charge().SubmitOTP(context.Background(), &paystack.ChargeSubmitOTPParams{Reference: "r", OTP: "1234"})
			return "/charge/submit_otp", err
		},
		"/charge/submit_phone": func(c paystack.ClientInterface) (string, error) {
			_, err := c.Charge().SubmitPhone(context.Background(), &paystack.ChargeSubmitPhoneParams{Reference: "r", Phone: "0800"})
			return "/charge/submit_phone", err
		},
		"/charge/submit_birthday": func(c paystack.ClientInterface) (string, error) {
			_, err := c.Charge().SubmitBirthday(context.Background(), &paystack.ChargeSubmitBirthdayParams{Reference: "r", Birthday: "2000-01-01"})
			return "/charge/submit_birthday", err
		},
	}
	for wantPath, fn := range paths {
		wantPath, fn := wantPath, fn
		t.Run(wantPath, func(t *testing.T) {
			mb := &testutil.MockBackend{}
			_, err := fn(newTestClient(t, mb))
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if mb.LastCall().Path != wantPath {
				t.Fatalf("path = %q, want %q", mb.LastCall().Path, wantPath)
			}
		})
	}
}

func TestCharge_CheckPending_CallShape(t *testing.T) {
	mb := &testutil.MockBackend{}
	c := newTestClient(t, mb)
	_, _ = c.Charge().CheckPending(context.Background(), "ref_x")
	if mb.LastCall().Path != "/charge/ref_x" {
		t.Fatalf("path = %q", mb.LastCall().Path)
	}
}

func TestCharge_ErrorMatrix(t *testing.T) {
	methods := map[string]func(paystack.ClientInterface) error{
		"Create": func(c paystack.ClientInterface) error {
			_, e := c.Charge().Create(context.Background(), &paystack.ChargeCreateParams{})
			return e
		},
		"SubmitPin": func(c paystack.ClientInterface) error {
			_, e := c.Charge().SubmitPin(context.Background(), &paystack.ChargeSubmitPinParams{})
			return e
		},
		"SubmitOTP": func(c paystack.ClientInterface) error {
			_, e := c.Charge().SubmitOTP(context.Background(), &paystack.ChargeSubmitOTPParams{})
			return e
		},
		"SubmitPhone": func(c paystack.ClientInterface) error {
			_, e := c.Charge().SubmitPhone(context.Background(), &paystack.ChargeSubmitPhoneParams{})
			return e
		},
		"SubmitBirthday": func(c paystack.ClientInterface) error {
			_, e := c.Charge().SubmitBirthday(context.Background(), &paystack.ChargeSubmitBirthdayParams{})
			return e
		},
		"CheckPending": func(c paystack.ClientInterface) error {
			_, e := c.Charge().CheckPending(context.Background(), "x")
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
