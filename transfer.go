package paystack

import (
	"context"
	"net/http"
	"net/url"
)

// Transferor is the contract for /transfer endpoints.
type Transferor interface {
	Initiate(ctx context.Context, params *TransferInitiateParams) (*Transfer, error)
	Finalize(ctx context.Context, params *TransferFinalizeParams) (*Transfer, error)
	Fetch(ctx context.Context, idOrCode string) (*Transfer, error)
	Verify(ctx context.Context, reference string) (*Transfer, error)
	List(ctx context.Context, params *TransferListParams) ([]Transfer, Meta, error)
}

// TransferService is the default Transferor implementation.
type TransferService struct {
	backend Backend
}

var _ Transferor = (*TransferService)(nil)

// Transfer is the Paystack transfer record.
type Transfer struct {
	ID            int64    `json:"id"`
	Domain        string   `json:"domain"`
	Amount        int64    `json:"amount"`
	Currency      string   `json:"currency"`
	Reference     string   `json:"reference"`
	Source        string   `json:"source"`
	Reason        string   `json:"reason"`
	Recipient     int64    `json:"recipient"`
	Status        string   `json:"status"`
	TransferCode  string   `json:"transfer_code"`
	CreatedAt     Time     `json:"createdAt"`
	UpdatedAt     Time     `json:"updatedAt"`
	FailureReason string   `json:"failure_reason"`
	Metadata      Metadata `json:"metadata,omitempty"`
}

// TransferInitiateParams is POST /transfer.
type TransferInitiateParams struct {
	Params
	Source    string  `json:"source"`
	Amount    int64   `json:"amount"`
	Recipient string  `json:"recipient"`
	Reason    *string `json:"reason,omitempty"`
	Currency  *string `json:"currency,omitempty"`
	Reference *string `json:"reference,omitempty"`
}

// TransferFinalizeParams is POST /transfer/finalize_transfer.
type TransferFinalizeParams struct {
	Params
	TransferCode string `json:"transfer_code"`
	OTP          string `json:"otp"`
}

// TransferListParams is GET /transfer.
type TransferListParams struct {
	ListParams
	Customer  *int64  `url:"customer,omitempty"`
	Recipient *int64  `url:"recipient,omitempty"`
	Status    *string `url:"status,omitempty"`
}

// Initiate queues a transfer to a recipient. A live-mode account may require
// a subsequent Finalize call if OTP is enabled.
func (s *TransferService) Initiate(ctx context.Context, p *TransferInitiateParams) (*Transfer, error) {
	var resp Response[Transfer]
	if err := s.backend.Call(ctx, http.MethodPost, "/transfer", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Finalize completes a two-step transfer by supplying the OTP.
func (s *TransferService) Finalize(ctx context.Context, p *TransferFinalizeParams) (*Transfer, error) {
	var resp Response[Transfer]
	if err := s.backend.Call(ctx, http.MethodPost, "/transfer/finalize_transfer", p, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Fetch returns a transfer by id or transfer_code.
func (s *TransferService) Fetch(ctx context.Context, idOrCode string) (*Transfer, error) {
	var resp Response[Transfer]
	if err := s.backend.Call(ctx, http.MethodGet, "/transfer/"+url.PathEscape(idOrCode), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Verify returns a transfer by reference.
func (s *TransferService) Verify(ctx context.Context, reference string) (*Transfer, error) {
	var resp Response[Transfer]
	if err := s.backend.Call(ctx, http.MethodGet, "/transfer/verify/"+url.PathEscape(reference), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// List returns a page of transfers.
func (s *TransferService) List(ctx context.Context, p *TransferListParams) ([]Transfer, Meta, error) {
	var resp ListResponse[Transfer]
	if err := s.backend.Call(ctx, http.MethodGet, "/transfer", p, &resp); err != nil {
		return nil, Meta{}, err
	}
	return resp.Data, resp.Meta, nil
}
