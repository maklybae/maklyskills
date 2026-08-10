//witness:dest services/orders/internal/api
//witness:red refund-bugs,refund-clean

package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"orderflow/services/orders/internal/api"
	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// Money that is on the order is reported as given back. The stock side of a
// refund may fail without the caller being told the refund did not happen.
func TestWitnessACommittedRefundIsNotAFailure(t *testing.T) {
	orders := testsupport.NewStore(seedOrder("ord_delivered", core.StatusDelivered))
	stock := testsupport.NewInventory()
	stock.FailRestock(errors.New("inventory is down"))

	service, err := core.NewService(orders, stock,
		core.WithClock(testsupport.Clock(placedAt)),
		core.WithIDSource(&testsupport.IDs{}),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	router, err := api.NewRouter(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	response := call(t, router, http.MethodPost, "/orders/ord_delivered/refund",
		`{"key":"sup-1","lines":[{"sku":"desk-lamp","quantity":1}]}`)

	stored, err := service.Get(context.Background(), "ord_delivered")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Refunded() == 0 {
		t.Skip("this build does not record the refund before the return goes out")
	}

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d for a refund of %d that the order already carries (body %s)",
			response.Code, stored.Refunded(), response.Body)
	}

	var view struct {
		ID     string `json:"id"`
		Amount int64  `json:"amount"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode body %q: %v", response.Body, err)
	}
	if view.ID == "" || view.Amount != stored.Refunded() {
		t.Fatalf("answer = %+v, want the refund that was recorded", view)
	}
}
