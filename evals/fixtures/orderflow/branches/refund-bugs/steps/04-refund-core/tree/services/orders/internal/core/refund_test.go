package core_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

func TestRefundOrder(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	tray := core.Item{SKU: "cable-tray", Quantity: 3, UnitPrice: 900}

	tests := []struct {
		name    string
		status  core.Status
		request core.RefundRequest
		wantErr error
	}{
		{
			name:    "the whole of a paid order",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-1"},
		},
		{
			name:   "one line of a delivered order",
			status: core.StatusDelivered,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-2",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
			},
		},
		{
			name:   "two lines of a shipped order",
			status: core.StatusShipped,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-3",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}, {SKU: "cable-tray", Quantity: 2}},
			},
		},
		{
			name:    "a cancelled order",
			status:  core.StatusCancelled,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-5"},
			wantErr: core.ErrNotRefundable,
		},
		{
			name:    "an order that is not there",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_missing", Key: "sup-6"},
			wantErr: core.ErrOrderNotFound,
		},
		{
			name:    "without an idempotency key",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1"},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "with a reason nobody will read",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-7", Reason: strings.Repeat("x", 501)},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "a sku the order never had",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-8",
				Lines:   []core.Item{{SKU: "hammock", Quantity: 1}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "more units than the order holds",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-9",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 3}},
			},
			wantErr: core.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := testsupport.NewStore(orderWith(tt.status, lamp, tray))
			service := newService(t, orders, testsupport.NewInventory())

			refund, err := service.RefundOrder(context.Background(), tt.request)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RefundOrder() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RefundOrder() error = %v", err)
			}

			if refund.Key != tt.request.Key {
				t.Errorf("key = %q, want %q", refund.Key, tt.request.Key)
			}
			if refund.ID == "" {
				t.Error("the refund came back without an id")
			}

			stored, err := service.Get(context.Background(), "ord_1")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if stored.Refunded() != refund.Amount {
				t.Errorf("the order records %d, the refund says %d", stored.Refunded(), refund.Amount)
			}
		})
	}
}

func TestRefundOrderRepeatsAnswerWithTheFirstRefund(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusPaid, lamp)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	request := core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}
	first, err := service.RefundOrder(context.Background(), request)
	if err != nil {
		t.Fatalf("first RefundOrder() error = %v", err)
	}
	second, err := service.RefundOrder(context.Background(), request)
	if err != nil {
		t.Fatalf("second RefundOrder() error = %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("second refund = %s, want the first one back (%s)", second.ID, first.ID)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Errorf("stored %d refunds, want 1", len(stored.Refunds))
	}
	if returns := stock.Returns(); len(returns) != 1 {
		t.Errorf("inventory saw %d returns, want 1", len(returns))
	}
}

func TestRefundOrderRecordsWhyAndWhen(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Reason:  "  lamp arrived scratched  ",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	if refund.Reason != "lamp arrived scratched" {
		t.Errorf("reason = %q, want it trimmed", refund.Reason)
	}
	if !refund.IssuedAt.Equal(now) {
		t.Errorf("issued at %s, want %s", refund.IssuedAt, now)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 || stored.Refunds[0].Reason != refund.Reason {
		t.Fatalf("stored refunds = %+v, want the one that was issued", stored.Refunds)
	}
}

func TestRefundOrderKeepsTheRefundWhenTheReturnFails(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
	stock.FailRestock(errors.New("inventory is down"))
	service := newService(t, orders, stock)

	request := core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}
	if _, err := service.RefundOrder(context.Background(), request); err == nil {
		t.Fatal("RefundOrder() = nil, want the inventory error")
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Fatalf("stored %d refunds, want the refund to survive the failed return", len(stored.Refunds))
	}
}

func TestRefundOrderKeepsWhatCameBack(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	tray := core.Item{SKU: "cable-tray", Quantity: 3, UnitPrice: 900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp, tray)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	lines := []core.Item{{SKU: "desk-lamp", Quantity: 1}, {SKU: "cable-tray", Quantity: 2}}
	refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   lines,
	})
	if err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	if len(refund.Lines) != 2 {
		t.Fatalf("the refund kept %d lines, want 2", len(refund.Lines))
	}
	for i, line := range refund.Lines {
		if line.SKU != lines[i].SKU || line.Quantity != lines[i].Quantity {
			t.Errorf("line %d = %+v, want %+v", i, line, lines[i])
		}
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 || len(stored.Refunds[0].Lines) != 2 {
		t.Fatalf("the order kept %+v, want the two lines that came back", stored.Refunds)
	}

	returns := stock.Returns()
	if len(returns) != 1 || len(returns[0].Lines) != 2 {
		t.Fatalf("inventory was told %+v, want one return of two lines", returns)
	}
	if returns[0].OrderID != "ord_1" {
		t.Errorf("the return names order %q, want ord_1", returns[0].OrderID)
	}
}

func TestRefundOrderTakesTheReasonAsItComes(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   string
	}{
		{name: "trimmed", reason: "  wrong colour  ", want: "wrong colour"},
		{name: "kept as written", reason: "wrong colour", want: "wrong colour"},
		{name: "none at all", reason: "", want: ""},
		{name: "a paragraph of it", reason: strings.Repeat("x", 120), want: strings.Repeat("x", 120)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
			orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp))
			service := newService(t, orders, testsupport.NewInventory())

			refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-1",
				Reason:  tt.reason,
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
			})
			if err != nil {
				t.Fatalf("RefundOrder() error = %v", err)
			}
			if refund.Reason != tt.want {
				t.Fatalf("reason = %q, want %q", refund.Reason, tt.want)
			}
		})
	}
}

func orderWith(status core.Status, items ...core.Item) core.Order {
	return core.Order{
		ID:         "ord_1",
		CustomerID: "cust-17",
		Items:      items,
		Status:     status,
		PlacedAt:   now.Add(-72 * time.Hour),
		UpdatedAt:  now.Add(-24 * time.Hour),
	}
}
