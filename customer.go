package paystack

import (
	"context"
	"net/http"
	"net/url"
)

// Customeror is the contract for /customer endpoints.
type Customeror interface {
	Create(ctx context.Context, params *CustomerCreateParams) (*Customer, error)
	Fetch(ctx context.Context, emailOrCode string) (*Customer, error)
	List(ctx context.Context, params *CustomerListParams) ([]Customer, Meta, error)
	Update(ctx context.Context, code string, params *CustomerUpdateParams) (*Customer, error)
	SetRiskAction(ctx context.Context, params *CustomerRiskActionParams) (*Customer, error)
	DeactivateAuthorization(ctx context.Context, authorizationCode string) error
}

// CustomerService is the default Customeror implementation.
type CustomerService struct {
	backend Backend
}

var _ Customeror = (*CustomerService)(nil)

// Customer is the full customer record returned by Paystack.
type Customer struct {
	ID              int64           `json:"id"`
	Domain          string          `json:"domain"`
	CustomerCode    string          `json:"customer_code"`
	Email           string          `json:"email"`
	FirstName       string          `json:"first_name"`
	LastName        string          `json:"last_name"`
	Phone           string          `json:"phone"`
	RiskAction      string          `json:"risk_action"`
	CreatedAt       Time            `json:"createdAt"`
	UpdatedAt       Time            `json:"updatedAt"`
	Metadata        Metadata        `json:"metadata,omitempty"`
	Authorizations  []Authorization `json:"authorizations,omitempty"`
	Transactions    []Transaction   `json:"transactions,omitempty"`
	Subscriptions   []Subscription  `json:"subscriptions,omitempty"`
	TotalVolume     int64           `json:"total_volume"`
	Identified      bool            `json:"identified"`
	IdentifiedValue Metadata        `json:"identifications,omitempty"`
}

// CustomerCreateParams is the payload for POST /customer.
type CustomerCreateParams struct {
	Params
	Email     string  `json:"email"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
}

// CustomerUpdateParams is the payload for PUT /customer/:code.
type CustomerUpdateParams struct {
	Params
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
}

// CustomerListParams is the query for GET /customer.
type CustomerListParams struct {
	ListParams
}

// CustomerRiskActionParams is the payload for POST /customer/set_risk_action.
type CustomerRiskActionParams struct {
	Params
	Customer   string `json:"customer"`
	RiskAction string `json:"risk_action"`
}

// Create registers a new customer.
func (s *CustomerService) Create(ctx context.Context, p *CustomerCreateParams) (*Customer, error) {
	var resp Response[Customer]
	if err := s.backend.Call(ctx, http.MethodPost, "/customer", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Fetch returns a customer by either email or customer_code.
func (s *CustomerService) Fetch(ctx context.Context, emailOrCode string) (*Customer, error) {
	var resp Response[Customer]
	if err := s.backend.Call(ctx, http.MethodGet, "/customer/"+url.PathEscape(emailOrCode), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// List returns a page of customers.
func (s *CustomerService) List(ctx context.Context, p *CustomerListParams) ([]Customer, Meta, error) {
	var resp ListResponse[Customer]
	if err := s.backend.Call(ctx, http.MethodGet, "/customer", p, &resp); err != nil {
		return nil, Meta{}, err
	}
	return resp.Data, resp.Meta, nil
}

// Update edits a customer identified by customer_code.
func (s *CustomerService) Update(ctx context.Context, code string, p *CustomerUpdateParams) (*Customer, error) {
	var resp Response[Customer]
	if err := s.backend.Call(ctx, http.MethodPut, "/customer/"+url.PathEscape(code), p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// SetRiskAction flags a customer as allow, deny, or default.
func (s *CustomerService) SetRiskAction(ctx context.Context, p *CustomerRiskActionParams) (*Customer, error) {
	var resp Response[Customer]
	if err := s.backend.Call(ctx, http.MethodPost, "/customer/set_risk_action", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeactivateAuthorization disables a saved authorization so it can no longer
// be charged.
func (s *CustomerService) DeactivateAuthorization(ctx context.Context, code string) error {
	body := struct {
		AuthorizationCode string `json:"authorization_code"`
	}{AuthorizationCode: code}
	return s.backend.Call(ctx, http.MethodPost, "/customer/deactivate_authorization", body, nil)
}
