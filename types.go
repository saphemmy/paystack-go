package paystack

import "encoding/json"

// CustomerLite is the reduced Customer shape embedded in other resources
// (transactions, subscriptions, refunds). Fetches of the full Customer record
// return the Customer type in customer.go.
type CustomerLite struct {
	ID           int64  `json:"id"`
	CustomerCode string `json:"customer_code"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
}

// Authorization is Paystack's saved-card representation. Returned embedded in
// Transaction, Charge, and Customer payloads whenever a reusable token
// exists.
type Authorization struct {
	AuthorizationCode string `json:"authorization_code"`
	Bin               string `json:"bin"`
	Last4             string `json:"last4"`
	ExpMonth          string `json:"exp_month"`
	ExpYear           string `json:"exp_year"`
	Channel           string `json:"channel"`
	CardType          string `json:"card_type"`
	Bank              string `json:"bank"`
	CountryCode       string `json:"country_code"`
	Brand             string `json:"brand"`
	Reusable          bool   `json:"reusable"`
	Signature         string `json:"signature"`
	AccountName       string `json:"account_name"`
}

// Metadata is the flexible blob Paystack echoes back on any resource that
// accepted Params.Metadata at creation time. Callers unmarshal it into their
// own struct.
type Metadata = json.RawMessage
