package paystack

import (
	"context"
	"net/http"
	"net/url"
)

// Charger is the contract for /charge endpoints.
type Charger interface {
	Create(ctx context.Context, params *ChargeCreateParams) (*ChargeResult, error)
	SubmitPin(ctx context.Context, params *ChargeSubmitPinParams) (*ChargeResult, error)
	SubmitOTP(ctx context.Context, params *ChargeSubmitOTPParams) (*ChargeResult, error)
	SubmitPhone(ctx context.Context, params *ChargeSubmitPhoneParams) (*ChargeResult, error)
	SubmitBirthday(ctx context.Context, params *ChargeSubmitBirthdayParams) (*ChargeResult, error)
	CheckPending(ctx context.Context, reference string) (*ChargeResult, error)
}

// ChargeService is the default Charger implementation.
type ChargeService struct {
	backend Backend
}

var _ Charger = (*ChargeService)(nil)

// ChargeResult is the progressively-updating charge state Paystack returns
// from every /charge endpoint. Callers check Status and, if further input is
// required (send_pin, send_otp, send_phone, send_birthday), prompt the
// customer then submit through the corresponding SubmitXxx method.
type ChargeResult struct {
	Reference       string         `json:"reference"`
	Status          string         `json:"status"`
	Display         string         `json:"display_text"`
	Message         string         `json:"message"`
	GatewayResponse string         `json:"gateway_response"`
	Amount          int64          `json:"amount"`
	Currency        string         `json:"currency"`
	Channel         string         `json:"channel"`
	PaidAt          Time           `json:"paid_at"`
	CreatedAt       Time           `json:"created_at"`
	Customer        *CustomerLite  `json:"customer,omitempty"`
	Authorization   *Authorization `json:"authorization,omitempty"`
	Metadata        Metadata       `json:"metadata,omitempty"`
}

// ChargeCreateParams is POST /charge. Provide one of authorization_code,
// bank, card, or mobile_money.
type ChargeCreateParams struct {
	Params
	Email             string      `json:"email"`
	Amount            int64       `json:"amount"`
	Currency          *string     `json:"currency,omitempty"`
	Reference         *string     `json:"reference,omitempty"`
	AuthorizationCode *string     `json:"authorization_code,omitempty"`
	PIN               *string     `json:"pin,omitempty"`
	Card              *ChargeCard `json:"card,omitempty"`
	Bank              *ChargeBank `json:"bank,omitempty"`
	MobileMoney       *ChargeMoMo `json:"mobile_money,omitempty"`
	Birthday          *string     `json:"birthday,omitempty"`
	DeviceID          *string     `json:"device_id,omitempty"`
}

// ChargeCard represents raw card details. Only used by merchants who have
// PCI scope; most callers should use authorization_code or the checkout flow.
type ChargeCard struct {
	Number   string `json:"number"`
	CVV      string `json:"cvv"`
	ExpMonth string `json:"expiry_month"`
	ExpYear  string `json:"expiry_year"`
}

// ChargeBank represents direct bank debit.
type ChargeBank struct {
	Code          string `json:"code"`
	AccountNumber string `json:"account_number"`
}

// ChargeMoMo represents mobile-money debit.
type ChargeMoMo struct {
	Phone    string `json:"phone"`
	Provider string `json:"provider"`
}

// ChargeSubmitPinParams is POST /charge/submit_pin.
type ChargeSubmitPinParams struct {
	Params
	Reference string `json:"reference"`
	PIN       string `json:"pin"`
}

// ChargeSubmitOTPParams is POST /charge/submit_otp.
type ChargeSubmitOTPParams struct {
	Params
	Reference string `json:"reference"`
	OTP       string `json:"otp"`
}

// ChargeSubmitPhoneParams is POST /charge/submit_phone.
type ChargeSubmitPhoneParams struct {
	Params
	Reference string `json:"reference"`
	Phone     string `json:"phone"`
}

// ChargeSubmitBirthdayParams is POST /charge/submit_birthday.
type ChargeSubmitBirthdayParams struct {
	Params
	Reference string `json:"reference"`
	Birthday  string `json:"birthday"`
}

// Create initiates a charge. Depending on Status the caller may need to
// invoke SubmitPin, SubmitOTP, SubmitPhone, or SubmitBirthday before the
// charge finalises.
func (s *ChargeService) Create(ctx context.Context, p *ChargeCreateParams) (*ChargeResult, error) {
	return s.post(ctx, "/charge", p)
}

// SubmitPin completes a charge that is awaiting a card PIN.
func (s *ChargeService) SubmitPin(ctx context.Context, p *ChargeSubmitPinParams) (*ChargeResult, error) {
	return s.post(ctx, "/charge/submit_pin", p)
}

// SubmitOTP completes a charge that is awaiting an OTP.
func (s *ChargeService) SubmitOTP(ctx context.Context, p *ChargeSubmitOTPParams) (*ChargeResult, error) {
	return s.post(ctx, "/charge/submit_otp", p)
}

// SubmitPhone completes a charge that is awaiting a phone number.
func (s *ChargeService) SubmitPhone(ctx context.Context, p *ChargeSubmitPhoneParams) (*ChargeResult, error) {
	return s.post(ctx, "/charge/submit_phone", p)
}

// SubmitBirthday completes a charge that is awaiting a birthday.
func (s *ChargeService) SubmitBirthday(ctx context.Context, p *ChargeSubmitBirthdayParams) (*ChargeResult, error) {
	return s.post(ctx, "/charge/submit_birthday", p)
}

// CheckPending re-polls a charge that remained in a pending state.
func (s *ChargeService) CheckPending(ctx context.Context, reference string) (*ChargeResult, error) {
	var resp Response[ChargeResult]
	if err := s.backend.Call(ctx, http.MethodGet, "/charge/"+url.PathEscape(reference), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (s *ChargeService) post(ctx context.Context, path string, params interface{}) (*ChargeResult, error) {
	var resp Response[ChargeResult]
	if err := s.backend.Call(ctx, http.MethodPost, path, params, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
