package paystack

// ClientInterface is the top-level contract implemented by the concrete
// *Client. Framework integration packages and application code should depend
// on this interface rather than the concrete type.
type ClientInterface interface {
	Transaction() Transactor
	Customer() Customeror
	Plan() Planner
	Subscription() Subscriber
	Transfer() Transferor
	Charge() Charger
	Refund() Refunder

	// Backend returns the underlying Backend. Useful for tests that swap it
	// at runtime, or for advanced callers issuing requests against endpoints
	// the SDK has not yet modelled.
	Backend() Backend
}

// Client is the concrete ClientInterface implementation. New returns it as
// ClientInterface; callers rarely need to reference this type directly.
type Client struct {
	backend      Backend
	transaction  *TransactionService
	customer     *CustomerService
	plan         *PlanService
	subscription *SubscriptionService
	transfer     *TransferService
	charge       *ChargeService
	refund       *RefundService
}

var _ ClientInterface = (*Client)(nil)

func newClient(key string, o *clientOptions) *Client {
	var b Backend
	if o.backend != nil {
		b = o.backend
	} else {
		b = NewHTTPBackend(key, &BackendConfig{
			HTTPClient:    o.httpClient,
			BaseURL:       o.baseURL,
			Logger:        o.logger,
			LeveledLogger: o.leveled,
		})
	}
	return &Client{
		backend:      b,
		transaction:  &TransactionService{backend: b},
		customer:     &CustomerService{backend: b},
		plan:         &PlanService{backend: b},
		subscription: &SubscriptionService{backend: b},
		transfer:     &TransferService{backend: b},
		charge:       &ChargeService{backend: b},
		refund:       &RefundService{backend: b},
	}
}

// Transaction returns the service for /transaction endpoints.
func (c *Client) Transaction() Transactor { return c.transaction }

// Customer returns the service for /customer endpoints.
func (c *Client) Customer() Customeror { return c.customer }

// Plan returns the service for /plan endpoints.
func (c *Client) Plan() Planner { return c.plan }

// Subscription returns the service for /subscription endpoints.
func (c *Client) Subscription() Subscriber { return c.subscription }

// Transfer returns the service for /transfer endpoints.
func (c *Client) Transfer() Transferor { return c.transfer }

// Charge returns the service for /charge endpoints.
func (c *Client) Charge() Charger { return c.charge }

// Refund returns the service for /refund endpoints.
func (c *Client) Refund() Refunder { return c.refund }

// Backend returns the underlying transport. Exposed for advanced callers.
func (c *Client) Backend() Backend { return c.backend }
