package core_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name       string
		seed       core.Status
		request    core.CancelRequest
		wantErr    error
		wantStatus core.Status
	}{
		{
			name:       "pending order is cancelled",
			seed:       core.StatusPending,
			request:    core.CancelRequest{OrderID: "ord_1", Reason: "customer changed their mind"},
			wantStatus: core.StatusCancelled,
		},
		{
			name:       "paid order is cancelled",
			seed:       core.StatusPaid,
			request:    core.CancelRequest{OrderID: "ord_1"},
			wantStatus: core.StatusCancelled,
		},
		{
			name:       "cancelling twice is a no-op",
			seed:       core.StatusCancelled,
			request:    core.CancelRequest{OrderID: "ord_1", Reason: "second attempt"},
			wantStatus: core.StatusCancelled,
		},
		{
			name:    "shipped order stays shipped",
			seed:    core.StatusShipped,
			request: core.CancelRequest{OrderID: "ord_1"},
			wantErr: core.ErrNotCancellable,
		},
		{
			name:    "delivered order stays delivered",
			seed:    core.StatusDelivered,
			request: core.CancelRequest{OrderID: "ord_1"},
			wantErr: core.ErrNotCancellable,
		},
		{
			name:    "unknown order",
			seed:    core.StatusPending,
			request: core.CancelRequest{OrderID: "ord_missing"},
			wantErr: core.ErrOrderNotFound,
		},
		{
			name:    "order id is required",
			seed:    core.StatusPending,
			request: core.CancelRequest{OrderID: "  "},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "reason has a limit",
			seed:    core.StatusPending,
			request: core.CancelRequest{OrderID: "ord_1", Reason: strings.Repeat("x", 241)},
			wantErr: core.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, stock := testsupport.NewStore(orderAt(tt.seed)), testsupport.NewInventory()
			service := newService(t, orders, stock)

			order, err := service.CancelOrder(context.Background(), tt.request)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CancelOrder() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CancelOrder() error = %v", err)
			}
			if order.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", order.Status, tt.wantStatus)
			}

			stored, err := service.Get(context.Background(), order.ID)
			if err != nil {
				t.Fatalf("Get(%s) error = %v", order.ID, err)
			}
			if stored.Status != tt.wantStatus {
				t.Fatalf("stored status = %s, want %s", stored.Status, tt.wantStatus)
			}
		})
	}
}

func TestCancelOrderRecordsWhyAndWhen(t *testing.T) {
	orders, stock := testsupport.NewStore(orderAt(core.StatusPaid)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	order, err := service.CancelOrder(context.Background(), core.CancelRequest{
		OrderID: "ord_1",
		Reason:  "  duplicate order  ",
	})
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}

	if order.CancelReason != "duplicate order" {
		t.Errorf("reason = %q, want %q", order.CancelReason, "duplicate order")
	}
	if order.CancelledAt == nil || !order.CancelledAt.Equal(now) {
		t.Errorf("cancelled at %v, want %s", order.CancelledAt, now)
	}
	if !order.UpdatedAt.Equal(now) {
		t.Errorf("updated at %s, want %s", order.UpdatedAt, now)
	}
}

func TestCancelOrderReleasesStock(t *testing.T) {
	orders, stock := testsupport.NewStore(orderAt(core.StatusPending)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	if _, err := service.CancelOrder(context.Background(), core.CancelRequest{OrderID: "ord_1", Reason: "wrong size"}); err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}

	releases := stock.Releases()
	if len(releases) != 1 {
		t.Fatalf("released %d reservations, want 1", len(releases))
	}
	if releases[0].OrderID != "ord_1" {
		t.Errorf("released order %s, want ord_1", releases[0].OrderID)
	}
	if releases[0].Reason != "wrong size" {
		t.Errorf("release reason = %q, want %q", releases[0].Reason, "wrong size")
	}
}

// A repeat of a cancel that failed halfway has to reach inventory again,
// otherwise the stock of that order stays reserved until someone notices.
func TestCancelOrderRepeatsTheRelease(t *testing.T) {
	orders, stock := testsupport.NewStore(orderAt(core.StatusPending)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	stock.FailRelease(errors.New("inventory is down"))
	if _, err := service.CancelOrder(context.Background(), core.CancelRequest{OrderID: "ord_1"}); err == nil {
		t.Fatal("CancelOrder() = nil, want the inventory error")
	}

	cancelled, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if cancelled.Status != core.StatusCancelled {
		t.Fatalf("status = %s, want %s: the state change must survive a failed release",
			cancelled.Status, core.StatusCancelled)
	}

	stock.FailRelease(nil)
	if _, err := service.CancelOrder(context.Background(), core.CancelRequest{OrderID: "ord_1"}); err != nil {
		t.Fatalf("second CancelOrder() error = %v", err)
	}
	if releases := stock.Releases(); len(releases) != 1 {
		t.Fatalf("inventory saw %d releases, want 1", len(releases))
	}
}

func TestCancelOrderKeepsStockWhenTheStoreFails(t *testing.T) {
	orders, stock := testsupport.NewStore(orderAt(core.StatusPaid)), testsupport.NewInventory()
	orders.FailUpdate(errors.New("disk full"))
	service := newService(t, orders, stock)

	_, err := service.CancelOrder(context.Background(), core.CancelRequest{OrderID: "ord_1"})
	if err == nil {
		t.Fatal("CancelOrder() = nil, want the store error")
	}
	if releases := stock.Releases(); len(releases) != 0 {
		t.Fatalf("released stock of an order that is still open (%d releases)", len(releases))
	}
}
