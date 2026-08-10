package core

// Level is what the warehouse holds for one sku.
type Level struct {
	SKU      string `json:"sku"`
	OnHand   int    `json:"on_hand"`
	Reserved int    `json:"reserved"`
}

// Line is a quantity of one sku, as it appears on an order.
type Line struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// Available is what can still be reserved.
func (l Level) Available() int {
	return l.OnHand - l.Reserved
}
