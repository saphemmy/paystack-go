package paystack

import (
	"context"
	"net/http"
	"net/url"
)

// Refunder is the contract for /refund endpoints.
type Refunder interface {
	Create(ctx context.Context, params *RefundCreateParams) (*Refund, error)
	Fetch(ctx context.Context, id string) (*Refund, error)
	List(ctx context.Context, params *RefundListParams) ([]Refund, Meta, error)
}

// RefundService is the default Refunder implementation.
type RefundService struct {
	backend Backend
}

var _ Refunder = (*RefundService)(nil)

// Refund is the Paystack refund record.
type Refund struct {
	ID             int64    `json:"id"`
	Domain         string   `json:"domain"`
	Transaction    int64    `json:"transaction"`
	Dispute        int64    `json:"dispute"`
	Amount         int64    `json:"amount"`
	DeductedAmount int64    `json:"deducted_amount"`
	Currency       string   `json:"currency"`
	Channel        string   `json:"channel"`
	MerchantNote   string   `json:"merchant_note"`
	CustomerNote   string   `json:"customer_note"`
	Status         string   `json:"status"`
	RefundedAt     Time     `json:"refunded_at"`
	RefundedBy     string   `json:"refunded_by"`
	ExpectedAt     Time     `json:"expected_at"`
	CreatedAt      Time     `json:"createdAt"`
	UpdatedAt      Time     `json:"updatedAt"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

// RefundCreateParams is POST /refund.
type RefundCreateParams struct {
	Params
	Transaction  string  `json:"transaction"`
	Amount       *int64  `json:"amount,omitempty"`
	Currency     *string `json:"currency,omitempty"`
	CustomerNote *string `json:"customer_note,omitempty"`
	MerchantNote *string `json:"merchant_note,omitempty"`
}

// RefundListParams is GET /refund.
type RefundListParams struct {
	ListParams
	Reference *string `url:"reference,omitempty"`
	Currency  *string `url:"currency,omitempty"`
}

// Create issues a new refund against a transaction.
func (s *RefundService) Create(ctx context.Context, p *RefundCreateParams) (*Refund, error) {
	var resp Response[Refund]
	if err := s.backend.Call(ctx, http.MethodPost, "/refund", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Fetch returns a refund by id.
func (s *RefundService) Fetch(ctx context.Context, id string) (*Refund, error) {
	var resp Response[Refund]
	if err := s.backend.Call(ctx, http.MethodGet, "/refund/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// List returns a page of refunds.
func (s *RefundService) List(ctx context.Context, p *RefundListParams) ([]Refund, Meta, error) {
	var resp ListResponse[Refund]
	if err := s.backend.Call(ctx, http.MethodGet, "/refund", p, &resp); err != nil {
		return nil, Meta{}, err
	}
	return resp.Data, resp.Meta, nil
}
