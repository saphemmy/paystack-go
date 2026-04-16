package paystack

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Transactor is the contract for /transaction endpoints.
type Transactor interface {
	Initialize(ctx context.Context, params *TransactionInitializeParams) (*TransactionInitializeData, error)
	Verify(ctx context.Context, reference string) (*Transaction, error)
	List(ctx context.Context, params *TransactionListParams) ([]Transaction, Meta, error)
	Fetch(ctx context.Context, id int64) (*Transaction, error)
	ChargeAuthorization(ctx context.Context, params *TransactionChargeAuthorizationParams) (*Transaction, error)
	Totals(ctx context.Context, params *TransactionTotalsParams) (*TransactionTotals, error)
}

// TransactionService is the default Transactor implementation.
type TransactionService struct {
	backend Backend
}

var _ Transactor = (*TransactionService)(nil)

// Transaction is the full Paystack transaction record.
type Transaction struct {
	ID              int64          `json:"id"`
	Domain          string         `json:"domain"`
	Status          string         `json:"status"`
	Reference       string         `json:"reference"`
	Amount          int64          `json:"amount"`
	Message         string         `json:"message"`
	GatewayResponse string         `json:"gateway_response"`
	PaidAt          Time           `json:"paid_at"`
	CreatedAt       Time           `json:"created_at"`
	Channel         string         `json:"channel"`
	Currency        string         `json:"currency"`
	IPAddress       string         `json:"ip_address"`
	Fees            int64          `json:"fees"`
	Metadata        Metadata       `json:"metadata,omitempty"`
	Customer        *CustomerLite  `json:"customer,omitempty"`
	Authorization   *Authorization `json:"authorization,omitempty"`
}

// TransactionInitializeParams is the payload for POST /transaction/initialize.
type TransactionInitializeParams struct {
	Params
	Email       string   `json:"email"`
	Amount      int64    `json:"amount"`
	Currency    *string  `json:"currency,omitempty"`
	Reference   *string  `json:"reference,omitempty"`
	CallbackURL *string  `json:"callback_url,omitempty"`
	Plan        *string  `json:"plan,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	SplitCode   *string  `json:"split_code,omitempty"`
	Subaccount  *string  `json:"subaccount,omitempty"`
	Bearer      *string  `json:"bearer,omitempty"`
}

// TransactionInitializeData is the response Data body for an initialize call.
type TransactionInitializeData struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

// TransactionListParams is the query for GET /transaction.
type TransactionListParams struct {
	ListParams
	Customer *int64  `url:"customer,omitempty"`
	Status   *string `url:"status,omitempty"`
	Amount   *int64  `url:"amount,omitempty"`
}

// TransactionChargeAuthorizationParams is POST /transaction/charge_authorization.
type TransactionChargeAuthorizationParams struct {
	Params
	Email             string   `json:"email"`
	Amount            int64    `json:"amount"`
	AuthorizationCode string   `json:"authorization_code"`
	Reference         *string  `json:"reference,omitempty"`
	Currency          *string  `json:"currency,omitempty"`
	Queue             *bool    `json:"queue,omitempty"`
	Channels          []string `json:"channels,omitempty"`
}

// TransactionTotalsParams is the query for GET /transaction/totals.
type TransactionTotalsParams struct {
	ListParams
}

// TransactionTotals is the aggregate returned by /transaction/totals.
type TransactionTotals struct {
	TotalTransactions     int64            `json:"total_transactions"`
	UniqueCustomers       int64            `json:"unique_customers"`
	TotalVolume           int64            `json:"total_volume"`
	TotalVolumeByCurrency []CurrencyVolume `json:"total_volume_by_currency"`
	PendingTransfers      int64            `json:"pending_transfers"`
}

// CurrencyVolume is one row of TransactionTotals.TotalVolumeByCurrency.
type CurrencyVolume struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

// Initialize begins a transaction and returns a checkout URL.
func (s *TransactionService) Initialize(ctx context.Context, p *TransactionInitializeParams) (*TransactionInitializeData, error) {
	var resp Response[TransactionInitializeData]
	if err := s.backend.Call(ctx, http.MethodPost, "/transaction/initialize", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Verify confirms the status of a transaction by reference.
func (s *TransactionService) Verify(ctx context.Context, reference string) (*Transaction, error) {
	var resp Response[Transaction]
	if err := s.backend.Call(ctx, http.MethodGet, "/transaction/verify/"+url.PathEscape(reference), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// List returns a page of transactions. Meta is always populated.
func (s *TransactionService) List(ctx context.Context, p *TransactionListParams) ([]Transaction, Meta, error) {
	var resp ListResponse[Transaction]
	if err := s.backend.Call(ctx, http.MethodGet, "/transaction", p, &resp); err != nil {
		return nil, Meta{}, err
	}
	return resp.Data, resp.Meta, nil
}

// Fetch returns a single transaction by numeric id.
func (s *TransactionService) Fetch(ctx context.Context, id int64) (*Transaction, error) {
	var resp Response[Transaction]
	if err := s.backend.Call(ctx, http.MethodGet, "/transaction/"+strconv.FormatInt(id, 10), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ChargeAuthorization charges a previously-saved authorization code.
func (s *TransactionService) ChargeAuthorization(ctx context.Context, p *TransactionChargeAuthorizationParams) (*Transaction, error) {
	var resp Response[Transaction]
	if err := s.backend.Call(ctx, http.MethodPost, "/transaction/charge_authorization", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Totals returns aggregate totals over the merchant account.
func (s *TransactionService) Totals(ctx context.Context, p *TransactionTotalsParams) (*TransactionTotals, error) {
	var resp Response[TransactionTotals]
	if err := s.backend.Call(ctx, http.MethodGet, "/transaction/totals", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
