package core

import "time"

// Item is one line of an order. Prices are in minor units to keep the
// arithmetic exact.
type Item struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

// Order is the aggregate this service owns.
type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Items      []Item    `json:"items"`
	Status     Status    `json:"status"`
	PlacedAt   time.Time `json:"placed_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Total is the order value in minor units.
func (o Order) Total() int64 {
	var total int64
	for _, item := range o.Items {
		total += item.UnitPrice * int64(item.Quantity)
	}
	return total
}

// Clone returns a copy that shares no memory with o, which is what stores hand
// out so a caller cannot reach into stored state.
func (o Order) Clone() Order {
	copied := o
	if o.Items != nil {
		copied.Items = make([]Item, len(o.Items))
		copy(copied.Items, o.Items)
	}
	return copied
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
