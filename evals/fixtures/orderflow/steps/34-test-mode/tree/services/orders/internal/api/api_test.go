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
