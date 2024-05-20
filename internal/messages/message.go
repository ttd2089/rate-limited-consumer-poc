package messages

// A Message is just a struct with some fields we can use as discriminators for the rate limiter.
type Message struct {
	CustomerID string `json:"customer_id"`
	Type       string `json:"type"`
	Body       string `json:"body"`
}
