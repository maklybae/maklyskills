package core_test

import (
	"context"
	"errors"
	"strings"
	"sync"
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
		want    int64
		wantErr error
	}{
		{
			name:    "the whole of a paid order",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-1"},
			want:    12500,
		},
		{
			name:   "one line of a delivered order",
			status: core.StatusDelivered,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-2",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
			},
			want: 4900,
		},
		{
			name:   "two lines of a shipped order",
			status: core.StatusShipped,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-3",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}, {SKU: "cable-tray", Quantity: 2}},
			},
			want: 6700,
		},
		{
			name:   "a reason of exactly the length we argued for",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-4",
				Reason:  strings.Repeat("x", 500),
			},
			want: 12500,
		},
		{
			name:    "an order nobody has paid for",
			status:  core.StatusPending,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-5"},
			wantErr: core.ErrNotRefundable,
		},
		{
			name:    "a cancelled order",
			status:  core.StatusCancelled,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-6"},
			wantErr: core.ErrNotRefundable,
		},
		{
			name:    "an order that is not there",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_missing", Key: "sup-7"},
			wantErr: core.ErrOrderNotFound,
		},
		{
			name:    "without an idempotency key",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1"},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "with a key that is a payload",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1", Key: strings.Repeat("k", 65)},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "with a reason nobody will read",
			status:  core.StatusPaid,
			request: core.RefundRequest{OrderID: "ord_1", Key: "sup-8", Reason: strings.Repeat("x", 501)},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "a line with no sku",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-9",
				Lines:   []core.Item{{Quantity: 1}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "a line of nothing",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-10",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 0}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "the same sku on two lines",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-11",
				Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}, {SKU: "desk-lamp", Quantity: 1}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "a sku the order never had",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-12",
				Lines:   []core.Item{{SKU: "hammock", Quantity: 1}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:   "more units than the order holds",
			status: core.StatusPaid,
			request: core.RefundRequest{
				OrderID: "ord_1",
				Key:     "sup-13",
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

			if refund.Amount != tt.want {
				t.Errorf("amount = %d, want %d", refund.Amount, tt.want)
			}
			if refund.Key != tt.request.Key {
				t.Errorf("key = %q, want %q", refund.Key, tt.request.Key)
			}

			stored, err := service.Get(context.Background(), "ord_1")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if stored.Refunded() != tt.want {
				t.Errorf("stored refunds = %d, want %d", stored.Refunded(), tt.want)
			}
		})
	}
}

// Two refunds of the same line take units out of the same pile.
func TestRefundOrderCountsUnitsAlreadySentBack(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}
	tray := core.Item{SKU: "cable-tray", Quantity: 3, UnitPrice: 900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp, tray)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	trays := []core.Item{{SKU: "cable-tray", Quantity: 3}}
	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-1", Lines: trays,
	}); err != nil {
		t.Fatalf("first RefundOrder() error = %v", err)
	}

	_, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-2", Lines: trays,
	})
	if !errors.Is(err, core.ErrInvalidRequest) {
		t.Fatalf("second RefundOrder() error = %v, want %v", err, core.ErrInvalidRequest)
	}

	returned := 0
	for _, ret := range stock.Returns() {
		for _, line := range ret.Lines {
			if line.SKU == "cable-tray" {
				returned += line.Quantity
			}
		}
	}
	if returned != 3 {
		t.Fatalf("inventory took back %d trays for an order that held 3", returned)
	}
}

func TestRefundOrderStopsAtWhatIsLeft(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	whole := core.RefundRequest{OrderID: "ord_1", Key: "sup-1"}
	if _, err := service.RefundOrder(context.Background(), whole); err != nil {
		t.Fatalf("first RefundOrder() error = %v", err)
	}

	line := core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-2",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}
	_, err := service.RefundOrder(context.Background(), line)
	if !errors.Is(err, core.ErrRefundTooLarge) {
		t.Fatalf("second RefundOrder() error = %v, want %v", err, core.ErrRefundTooLarge)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Refunded() != 9800 {
		t.Fatalf("refunded %d in total, want 9800", stored.Refunded())
	}
}

func TestRefundOrderRefusesARefundOfNothing(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusPaid, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-1",
	}); err != nil {
		t.Fatalf("first RefundOrder() error = %v", err)
	}

	_, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-2", Reason: strings.Repeat("x", 500),
	})
	if !errors.Is(err, core.ErrInvalidRequest) {
		t.Fatalf("RefundOrder() error = %v, want %v", err, core.ErrInvalidRequest)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Fatalf("the order holds %d refunds, want the one that moved money", len(stored.Refunds))
	}
}

func TestRefundOrderRepeatsAnswerWithTheFirstRefund(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
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

	if first.ID != second.ID || second.Amount != 4900 {
		t.Fatalf("second refund = %+v, want the first one back (%+v)", second, first)
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

// Nothing left the warehouse for an order that was only paid for, so there is
// nothing to shelve; what the warehouse holds for it stays held until a cancel
// gives it back.
func TestRefundOrderShelvesNothingThatNeverLeft(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusPaid, lamp)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	if returns := stock.Returns(); len(returns) != 0 {
		t.Fatalf("inventory shelved %d returns of goods that never left", len(returns))
	}
	if releases := stock.Releases(); len(releases) != 0 {
		t.Fatalf("inventory saw %d releases, want the hold left to the cancel path", len(releases))
	}
}

func TestRefundOrderSendsTheLinesBackOnce(t *testing.T) {
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

	// A return inventory may have already booked must not be sent again.
	if returns := stock.Returns(); len(returns) != 1 {
		t.Fatalf("inventory saw %d returns, want 1", len(returns))
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Refunded() != 4900 {
		t.Fatalf("refunded %d, want the refund to survive the failed return", stored.Refunded())
	}
}

// The money is committed before the return is sent, so the return is not the
// caller's to cancel.
func TestRefundOrderReturnsStockAfterTheCallerHangsUp(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.RefundOrder(ctx, core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	if returns := stock.Returns(); len(returns) != 1 {
		t.Fatalf("inventory saw %d returns, want the committed return to go out anyway", len(returns))
	}
}

// One key, many callers: support clicking twice must not pay twice.
func TestRefundOrderIsSerialisedPerOrder(t *testing.T) {
	const callers = 16

	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	request := core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}

	var start, done sync.WaitGroup
	start.Add(1)
	for range callers {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			if _, err := service.RefundOrder(context.Background(), request); err != nil {
				t.Errorf("RefundOrder() error = %v", err)
			}
		}()
	}
	start.Done()
	done.Wait()

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Errorf("the order holds %d refunds for one key, want 1", len(stored.Refunds))
	}
	if stored.Refunded() != 4900 {
		t.Errorf("refunded %d for one key, want 4900", stored.Refunded())
	}
	if returns := stock.Returns(); len(returns) != 1 {
		t.Errorf("inventory saw %d returns for one refund, want 1", len(returns))
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

// Goods that have been paid back do not leave the building.
func TestRefundedOrderDoesNotShip(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusPaid, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-1",
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	_, err := service.MarkShipped(context.Background(), "ord_1")
	if !errors.Is(err, core.ErrNotShippable) {
		t.Fatalf("MarkShipped() error = %v, want %v", err, core.ErrNotShippable)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Status != core.StatusPaid {
		t.Fatalf("status = %s, want the order to stay where it was", stored.Status)
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
