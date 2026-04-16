package paystack

// Params is the base embedded into every request struct. It carries values
// that are sent as HTTP headers or top-level metadata, never as plain body
// fields.
type Params struct {
	// IdempotencyKey, when set, is forwarded on the Idempotency-Key header
	// for writes. The SDK never generates one automatically.
	IdempotencyKey *string `json:"-"`

	// Metadata attaches an arbitrary JSON object to a resource. Paystack
	// echoes it back on fetches and webhooks.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListParams is embedded in list-endpoint request structs. All fields are
// optional; nil leaves the parameter off the query string entirely.
type ListParams struct {
	Params  `json:"-" url:"-"`
	PerPage *int  `json:"-" url:"perPage,omitempty"`
	Page    *int  `json:"-" url:"page,omitempty"`
	From    *Time `json:"-" url:"from,omitempty"`
	To      *Time `json:"-" url:"to,omitempty"`
}

// Meta is the pagination envelope Paystack returns on list endpoints.
// It is exposed raw — the SDK never hides pagination state.
type Meta struct {
	Total     int `json:"total"`
	Skipped   int `json:"skipped"`
	PerPage   int `json:"perPage"`
	Page      int `json:"page"`
	PageCount int `json:"pageCount"`
}

// ListResponse is the generic envelope every Paystack list endpoint returns.
// An empty Data slice with Meta.Total > 0 is a legitimate response (the caller
// paged past the end) and is never treated as an error.
type ListResponse[T any] struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    []T    `json:"data"`
	Meta    Meta   `json:"meta"`
}

// Response is the single-record envelope Paystack returns from non-list
// endpoints. Callers typically do not see this — the client unwraps Data
// into the caller-supplied struct.
type Response[T any] struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// paramCarrier is satisfied by any request struct that embeds Params. The
// backend uses it to pull the idempotency key out without reflecting over
// every request type.
type paramCarrier interface {
	paystackParams() Params
}

func (p Params) paystackParams() Params { return p }
