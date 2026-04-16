package paystack

import (
	"context"
	"net/http"
	"net/url"
)

// Subscriber is the contract for /subscription endpoints.
type Subscriber interface {
	Create(ctx context.Context, params *SubscriptionCreateParams) (*Subscription, error)
	Fetch(ctx context.Context, idOrCode string) (*Subscription, error)
	List(ctx context.Context, params *SubscriptionListParams) ([]Subscription, Meta, error)
	Enable(ctx context.Context, params *SubscriptionToggleParams) error
	Disable(ctx context.Context, params *SubscriptionToggleParams) error
	GenerateUpdateLink(ctx context.Context, code string) (*SubscriptionManageLink, error)
}

// SubscriptionService is the default Subscriber implementation.
type SubscriptionService struct {
	backend Backend
}

var _ Subscriber = (*SubscriptionService)(nil)

// Subscription is the Paystack subscription record.
type Subscription struct {
	ID               int64          `json:"id"`
	Domain           string         `json:"domain"`
	Status           string         `json:"status"`
	SubscriptionCode string         `json:"subscription_code"`
	EmailToken       string         `json:"email_token"`
	Amount           int64          `json:"amount"`
	CronExpression   string         `json:"cron_expression"`
	NextPaymentDate  Time           `json:"next_payment_date"`
	OpenInvoice      string         `json:"open_invoice"`
	CreatedAt        Time           `json:"createdAt"`
	UpdatedAt        Time           `json:"updatedAt"`
	Customer         *CustomerLite  `json:"customer,omitempty"`
	Plan             *Plan          `json:"plan,omitempty"`
	Authorization    *Authorization `json:"authorization,omitempty"`
}

// SubscriptionCreateParams is POST /subscription.
type SubscriptionCreateParams struct {
	Params
	Customer      string  `json:"customer"`
	Plan          string  `json:"plan"`
	Authorization *string `json:"authorization,omitempty"`
	StartDate     *Time   `json:"start_date,omitempty"`
}

// SubscriptionToggleParams is POST /subscription/enable and /disable.
type SubscriptionToggleParams struct {
	Params
	Code  string `json:"code"`
	Token string `json:"token"`
}

// SubscriptionListParams is GET /subscription.
type SubscriptionListParams struct {
	ListParams
	Customer *int64  `url:"customer,omitempty"`
	Plan     *int64  `url:"plan,omitempty"`
	Status   *string `url:"status,omitempty"`
}

// SubscriptionManageLink is the response of /subscription/:code/manage/link.
type SubscriptionManageLink struct {
	Link string `json:"link"`
}

// Create initiates a new subscription.
func (s *SubscriptionService) Create(ctx context.Context, p *SubscriptionCreateParams) (*Subscription, error) {
	var resp Response[Subscription]
	if err := s.backend.Call(ctx, http.MethodPost, "/subscription", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Fetch returns a subscription by id or subscription_code.
func (s *SubscriptionService) Fetch(ctx context.Context, idOrCode string) (*Subscription, error) {
	var resp Response[Subscription]
	if err := s.backend.Call(ctx, http.MethodGet, "/subscription/"+url.PathEscape(idOrCode), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// List returns a page of subscriptions.
func (s *SubscriptionService) List(ctx context.Context, p *SubscriptionListParams) ([]Subscription, Meta, error) {
	var resp ListResponse[Subscription]
	if err := s.backend.Call(ctx, http.MethodGet, "/subscription", p, &resp); err != nil {
		return nil, Meta{}, err
	}
	return resp.Data, resp.Meta, nil
}

// Enable re-activates a previously-disabled subscription.
func (s *SubscriptionService) Enable(ctx context.Context, p *SubscriptionToggleParams) error {
	return s.backend.Call(ctx, http.MethodPost, "/subscription/enable", p, nil)
}

// Disable halts further billing on an active subscription.
func (s *SubscriptionService) Disable(ctx context.Context, p *SubscriptionToggleParams) error {
	return s.backend.Call(ctx, http.MethodPost, "/subscription/disable", p, nil)
}

// GenerateUpdateLink returns a hosted URL where the subscriber can update
// their payment method.
func (s *SubscriptionService) GenerateUpdateLink(ctx context.Context, code string) (*SubscriptionManageLink, error) {
	var resp Response[SubscriptionManageLink]
	if err := s.backend.Call(ctx, http.MethodGet, "/subscription/"+url.PathEscape(code)+"/manage/link", nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
