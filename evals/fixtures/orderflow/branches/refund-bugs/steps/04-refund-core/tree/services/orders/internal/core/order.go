package core

import (
	"slices"
	"time"
)

// Item is one line of an order. Prices are in minor units to keep the
// arithmetic exact.
type Item struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_amount"`
}

// Order is the aggregate this service owns.
type Order struct {
	ID           string     `json:"id"`
	CustomerID   string     `json:"customer_id"`
	Items        []Item     `json:"items"`
	Status       Status     `json:"status"`
	PlacedAt     time.Time  `json:"placed_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`
	CancelReason string     `json:"cancel_reason,omitempty"`
	Refunds      []Refund   `json:"refunds,omitempty"`
}

// Refund is money given back for part or all of an order.
type Refund struct {
	ID       string    `json:"id"`
	Key      string    `json:"key"`
	Amount   int64     `json:"amount"`
	Reason   string    `json:"reason,omitempty"`
	Lines    []Item    `json:"lines"`
	IssuedAt time.Time `json:"issued_at"`
}

// Total is what the order is worth in minor units, less what has been
// refunded.
func (o Order) Total() int64 {
	return linesValue(o.Items) - o.Refunded()
}

// Refunded is what has already been given back, in minor units.
func (o Order) Refunded() int64 {
	var refunded int64
	for _, refund := range o.Refunds {
		refunded += refund.Amount
	}
	return refunded
}

// Clone returns a copy that shares no memory with o, which is what stores hand
// out so a caller cannot reach into stored state.
func (o Order) Clone() Order {
	copied := o
	if o.Items != nil {
		copied.Items = make([]Item, len(o.Items))
		copy(copied.Items, o.Items)
	}
	if o.CancelledAt != nil {
		at := *o.CancelledAt
		copied.CancelledAt = &at
	}
	if o.Refunds != nil {
		copied.Refunds = make([]Refund, len(o.Refunds))
		for i, refund := range o.Refunds {
			copied.Refunds[i] = refund
			copied.Refunds[i].Lines = slices.Clone(refund.Lines)
		}
	}
	return copied
}

func linesValue(items []Item) int64 {
	var value int64
	for _, item := range items {
		value += item.UnitPrice * int64(item.Quantity)
	}
	return value
}

func (i Item) validate() error {
	switch {
	case i.SKU == "":
		return invalid("items", "every item needs a sku")
	case i.Quantity <= 0:
		return invalid("items", "quantity of "+i.SKU+" must be positive")
	case i.UnitPrice < 0:
		return invalid("items", "unit price of "+i.SKU+" must not be negative")
	}
	return nil
}
