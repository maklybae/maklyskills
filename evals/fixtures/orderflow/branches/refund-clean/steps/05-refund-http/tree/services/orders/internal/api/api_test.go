package api_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orderflow/pkg/httputil"
	"orderflow/services/orders/internal/api"
	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

var placedAt = time.Date(2024, time.July, 9, 8, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	testsupport.Main(m)
}

func TestEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "place an order",
			method:     http.MethodPost,
			path:       "/orders",
			body:       `{"customer_id":"cust-17","items":[{"sku":"desk-lamp","quantity":1,"unit_price":4900}]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "place an order without items",
			method:     http.MethodPost,
			path:       "/orders",
			body:       `{"customer_id":"cust-17","items":[]}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "place an order with a misspelled field",
			method:     http.MethodPost,
			path:       "/orders",
			body:       `{"custmer_id":"cust-17"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "fetch an order",
			method:     http.MethodGet,
			path:       "/orders/ord_open",
			wantStatus: http.StatusOK,
		},
		{
			name:       "fetch an unknown order",
			method:     http.MethodGet,
			path:       "/orders/ord_missing",
			wantStatus: http.StatusNotFound,
			wantCode:   "order_not_found",
		},
		{
			name:       "list orders",
			method:     http.MethodGet,
			path:       "/orders?customer_id=cust-17&status=pending",
			wantStatus: http.StatusOK,
		},
		{
			name:       "list with an unknown status",
			method:     http.MethodGet,
			path:       "/orders?status=lost",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "cancel an order",
			method:     http.MethodPost,
			path:       "/orders/ord_open/cancel",
			body:       `{"reason":"customer changed their mind"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "cancel without a reason",
			method:     http.MethodPost,
			path:       "/orders/ord_open/cancel",
			wantStatus: http.StatusOK,
		},
		{
			name:       "cancel a shipped order",
			method:     http.MethodPost,
			path:       "/orders/ord_shipped/cancel",
			wantStatus: http.StatusConflict,
			wantCode:   "order_not_cancellable",
		},
		{
			name:       "refund a delivered order",
			method:     http.MethodPost,
			path:       "/orders/ord_delivered/refund",
			body:       `{"key":"sup-1","reason":"lamp arrived scratched","lines":[{"sku":"desk-lamp","quantity":1}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "refund without an idempotency key",
			method:     http.MethodPost,
			path:       "/orders/ord_delivered/refund",
			body:       `{"reason":"lamp arrived scratched"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "refund an order nobody paid for",
			method:     http.MethodPost,
			path:       "/orders/ord_open/refund",
			body:       `{"key":"sup-2"}`,
			wantStatus: http.StatusConflict,
			wantCode:   "order_not_refundable",
		},
		{
			name:       "refund more units than the order holds",
			method:     http.MethodPost,
			path:       "/orders/ord_delivered/refund",
			body:       `{"key":"sup-3","lines":[{"sku":"desk-lamp","quantity":9}]}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "cancel an unknown order",
			method:     http.MethodPost,
			path:       "/orders/ord_missing/cancel",
			wantStatus: http.StatusNotFound,
			wantCode:   "order_not_found",
		},
		{
			name:       "ship an order that is not paid",
			method:     http.MethodPost,
			path:       "/orders/ord_open/ship",
			wantStatus: http.StatusConflict,
			wantCode:   "order_not_shippable",
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/invoices",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := newRouter(t)

			response := call(t, router, tt.method, tt.path, tt.body)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", response.Code, tt.wantStatus, response.Body)
			}
			if tt.wantCode == "" {
				return
			}

			var failure httputil.ErrorBody
			if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
				t.Fatalf("decode error body %q: %v", response.Body, err)
			}
			if failure.Error.Code != tt.wantCode {
				t.Fatalf("error code = %q, want %q", failure.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestCancelIsIdempotentOverHTTP(t *testing.T) {
	router, _ := newRouter(t)

	first := call(t, router, http.MethodPost, "/orders/ord_open/cancel", `{"reason":"duplicate order"}`)
	second := call(t, router, http.MethodPost, "/orders/ord_open/cancel", `{"reason":"duplicate order"}`)

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d and %d, want 200 twice", first.Code, second.Code)
	}

	var view struct {
		Status       string `json:"status"`
		CancelReason string `json:"cancel_reason"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode body %q: %v", second.Body, err)
	}
	if view.Status != "cancelled" {
		t.Errorf("status = %q, want %q", view.Status, "cancelled")
	}
	if view.CancelReason != "duplicate order" {
		t.Errorf("reason = %q, want %q", view.CancelReason, "duplicate order")
	}
}

func TestRefundAnswersWithTheRefund(t *testing.T) {
	router, _ := newRouter(t)

	response := call(t, router, http.MethodPost, "/orders/ord_delivered/refund",
		`{"key":"sup-1","reason":"lamp arrived scratched","lines":[{"sku":"desk-lamp","quantity":1}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", response.Code, response.Body)
	}

	var view struct {
		ID      string `json:"id"`
		OrderID string `json:"order_id"`
		Amount  int64  `json:"amount"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode body %q: %v", response.Body, err)
	}
	if view.Amount != 4900 {
		t.Errorf("amount = %d, want 4900", view.Amount)
	}
	if view.OrderID != "ord_delivered" || view.ID == "" {
		t.Errorf("refund = %+v, want it to name the order and itself", view)
	}
	if view.Reason != "lamp arrived scratched" {
		t.Errorf("reason = %q, want the one that was sent", view.Reason)
	}

	// The order the refund belongs to still reports what it was worth, with what
	// has gone back beside it rather than taken out of it.
	order := call(t, router, http.MethodGet, "/orders/ord_delivered", "")
	var stored struct {
		Total    int64 `json:"total"`
		Refunded int64 `json:"refunded"`
	}
	if err := json.Unmarshal(order.Body.Bytes(), &stored); err != nil {
		t.Fatalf("decode order %q: %v", order.Body, err)
	}
	if stored.Total != 4900 {
		t.Errorf("total = %d, want the order value 4900", stored.Total)
	}
	if stored.Refunded != 4900 {
		t.Errorf("refunded = %d, want 4900", stored.Refunded)
	}
}

func TestRefundBeyondTheCeilingIsAConflict(t *testing.T) {
	router, _ := newRouter(t)

	first := call(t, router, http.MethodPost, "/orders/ord_delivered/refund", `{"key":"sup-1"}`)
	if first.Code != http.StatusOK {
		t.Fatalf("first refund status = %d, want 200 (body %s)", first.Code, first.Body)
	}

	second := call(t, router, http.MethodPost, "/orders/ord_delivered/refund",
		`{"key":"sup-2","lines":[{"sku":"desk-lamp","quantity":1}]}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("second refund status = %d, want 409 (body %s)", second.Code, second.Body)
	}

	var failure httputil.ErrorBody
	if err := json.Unmarshal(second.Body.Bytes(), &failure); err != nil {
		t.Fatalf("decode error body %q: %v", second.Body, err)
	}
	if failure.Error.Code != "refund_too_large" {
		t.Fatalf("error code = %q, want %q", failure.Error.Code, "refund_too_large")
	}
}

func TestStoreFailureIsNotTheClientsFault(t *testing.T) {
	router, orders := newRouter(t)
	orders.FailGet(errors.New("disk on fire"))

	response := call(t, router, http.MethodGet, "/orders/ord_open", "")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	var failure httputil.ErrorBody
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatalf("decode error body %q: %v", response.Body, err)
	}
	if failure.Error.Message != "internal error" {
		t.Fatalf("message = %q, want %q: internals do not belong in a response",
			failure.Error.Message, "internal error")
	}
}

func newRouter(t *testing.T) (http.Handler, *testsupport.Store) {
	t.Helper()

	orders := testsupport.NewStore(
		seedOrder("ord_open", core.StatusPending),
		seedOrder("ord_shipped", core.StatusShipped),
		seedOrder("ord_delivered", core.StatusDelivered),
	)
	service, err := core.NewService(orders, testsupport.NewInventory(),
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
	return router, orders
}

func call(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func seedOrder(id string, status core.Status) core.Order {
	return core.Order{
		ID:         id,
		CustomerID: "cust-17",
		Items:      []core.Item{{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}},
		Status:     status,
		PlacedAt:   placedAt,
		UpdatedAt:  placedAt,
	}
}
