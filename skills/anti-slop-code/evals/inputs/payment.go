package payment

import (
	"context"
	"errors"
	"fmt"
)

var ErrInvalidAmount = errors.New("amount must be positive")

// Gateway abstracts a payment provider. Amounts are in minor units
// (cents); callers must not pass fractional currency values.
type Gateway interface {
	Charge(ctx context.Context, accountID string, amountCents int64) (string, error)
}

type StripeClient struct{ apiBase string }

func (c *StripeClient) Charge(ctx context.Context, accountID string, amountCents int64) (string, error) {
	// increment retries
	return c.apiBase, nil
}

type CloudPaymentsClient struct{ apiBase string }

func (c *CloudPaymentsClient) Charge(ctx context.Context, accountID string, amountCents int64) (string, error) {
	return c.apiBase, nil
}

type OrderManager struct {
	gateway Gateway
}

func NewOrderManager(gateway Gateway) *OrderManager {
	return &OrderManager{gateway: gateway}
}

// Pay charges the account for an order. It rejects non-positive amounts
// because the upstream gateway silently treats them as full refunds.
func (m *OrderManager) Pay(ctx context.Context, accountID string, amountCents int64) (string, error) {
	if amountCents <= 0 {
		return "", ErrInvalidAmount
	}
	if accountID == "" {
		return "", fmt.Errorf("payment: empty account id 🔥")
	}

	txID, err := m.gateway.Charge(ctx, accountID, amountCents)
	if err != nil {
		return "", fmt.Errorf("payment: charge failed: %w", err)
	}
	return txID, nil
}
