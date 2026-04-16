package paystack

import (
	"context"
	"net/http"
	"net/url"
)

// Planner is the contract for /plan endpoints.
type Planner interface {
	Create(ctx context.Context, params *PlanCreateParams) (*Plan, error)
	Fetch(ctx context.Context, idOrCode string) (*Plan, error)
	List(ctx context.Context, params *PlanListParams) ([]Plan, Meta, error)
	Update(ctx context.Context, idOrCode string, params *PlanUpdateParams) error
}

// PlanService is the default Planner implementation.
type PlanService struct {
	backend Backend
}

var _ Planner = (*PlanService)(nil)

// Plan is the Paystack subscription plan record.
type Plan struct {
	ID                int64    `json:"id"`
	Name              string   `json:"name"`
	PlanCode          string   `json:"plan_code"`
	Description       string   `json:"description"`
	Amount            int64    `json:"amount"`
	Interval          string   `json:"interval"`
	SendInvoices      bool     `json:"send_invoices"`
	SendSMS           bool     `json:"send_sms"`
	HostedPage        bool     `json:"hosted_page"`
	HostedPageURL     string   `json:"hosted_page_url"`
	HostedPageSummary string   `json:"hosted_page_summary"`
	Currency          string   `json:"currency"`
	InvoiceLimit      int      `json:"invoice_limit"`
	CreatedAt         Time     `json:"createdAt"`
	UpdatedAt         Time     `json:"updatedAt"`
	Metadata          Metadata `json:"metadata,omitempty"`
}

// PlanCreateParams is POST /plan.
type PlanCreateParams struct {
	Params
	Name         string  `json:"name"`
	Amount       int64   `json:"amount"`
	Interval     string  `json:"interval"`
	Description  *string `json:"description,omitempty"`
	SendInvoices *bool   `json:"send_invoices,omitempty"`
	SendSMS      *bool   `json:"send_sms,omitempty"`
	Currency     *string `json:"currency,omitempty"`
	InvoiceLimit *int    `json:"invoice_limit,omitempty"`
}

// PlanUpdateParams is PUT /plan/:id_or_code.
type PlanUpdateParams struct {
	Params
	Name         *string `json:"name,omitempty"`
	Amount       *int64  `json:"amount,omitempty"`
	Interval     *string `json:"interval,omitempty"`
	Description  *string `json:"description,omitempty"`
	SendInvoices *bool   `json:"send_invoices,omitempty"`
	SendSMS      *bool   `json:"send_sms,omitempty"`
	Currency     *string `json:"currency,omitempty"`
	InvoiceLimit *int    `json:"invoice_limit,omitempty"`
}

// PlanListParams is GET /plan.
type PlanListParams struct {
	ListParams
	Status   *string `url:"status,omitempty"`
	Interval *string `url:"interval,omitempty"`
	Amount   *int64  `url:"amount,omitempty"`
}

// Create registers a new plan.
func (s *PlanService) Create(ctx context.Context, p *PlanCreateParams) (*Plan, error) {
	var resp Response[Plan]
	if err := s.backend.Call(ctx, http.MethodPost, "/plan", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Fetch returns a plan by id or code.
func (s *PlanService) Fetch(ctx context.Context, idOrCode string) (*Plan, error) {
	var resp Response[Plan]
	if err := s.backend.Call(ctx, http.MethodGet, "/plan/"+url.PathEscape(idOrCode), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// List returns a page of plans.
func (s *PlanService) List(ctx context.Context, p *PlanListParams) ([]Plan, Meta, error) {
	var resp ListResponse[Plan]
	if err := s.backend.Call(ctx, http.MethodGet, "/plan", p, &resp); err != nil {
		return nil, Meta{}, err
	}
	return resp.Data, resp.Meta, nil
}

// Update edits a plan. Paystack returns no Data body, so Update returns only
// an error.
func (s *PlanService) Update(ctx context.Context, idOrCode string, p *PlanUpdateParams) error {
	return s.backend.Call(ctx, http.MethodPut, "/plan/"+url.PathEscape(idOrCode), p, nil)
}
