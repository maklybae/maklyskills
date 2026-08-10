package api_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orderflow/pkg/httputil"
	"orderflow/services/inventory/internal/api"
	"orderflow/services/inventory/internal/core"
	"orderflow/services/inventory/internal/store"
)

func TestEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "health", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK},
		{name: "the whole shelf", method: http.MethodGet, path: "/stock", wantStatus: http.StatusOK},
		{name: "one sku", method: http.MethodGet, path: "/stock/desk-lamp", wantStatus: http.StatusOK},
		{
			name:       "unknown sku",
			method:     http.MethodGet,
			path:       "/stock/hammock",
			wantStatus: http.StatusNotFound,
			wantCode:   "unknown_sku",
		},
		{
			name:       "reserve",
			method:     http.MethodPost,
			path:       "/reservations",
			body:       `{"order_id":"ord_1","lines":[{"sku":"desk-lamp","quantity":2}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "reserve more than the shelf holds",
			method:     http.MethodPost,
			path:       "/reservations",
			body:       `{"order_id":"ord_1","lines":[{"sku":"monitor-arm","quantity":99}]}`,
			wantStatus: http.StatusConflict,
			wantCode:   "out_of_stock",
		},
		{
			name:       "reserve without an order id",
			method:     http.MethodPost,
			path:       "/reservations",
			body:       `{"lines":[{"sku":"desk-lamp","quantity":2}]}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "release an unknown order",
			method:     http.MethodPost,
			path:       "/reservations/release",
			body:       `{"order_id":"ord_unknown"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a return",
			method:     http.MethodPost,
			path:       "/stock/returns",
			body:       `{"order_id":"ord_1","lines":[{"sku":"desk-lamp","quantity":2}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a return of nothing",
			method:     http.MethodPost,
			path:       "/stock/returns",
			body:       `{"order_id":"ord_1","lines":[]}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "restock",
			method:     http.MethodPost,
			path:       "/stock/desk-lamp/restock",
			body:       `{"quantity":10}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "restock with a broken body",
			method:     http.MethodPost,
			path:       "/stock/desk-lamp/restock",
			body:       `{"quantity":}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t)

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

func TestReserveThenRelease(t *testing.T) {
	router := newRouter(t)

	reserve := call(t, router, http.MethodPost, "/reservations", `{"order_id":"ord_1","lines":[{"sku":"monitor-arm","quantity":3}]}`)
	if reserve.Code != http.StatusOK {
		t.Fatalf("reserve status = %d, want 200 (body %s)", reserve.Code, reserve.Body)
	}

	afterReserve := levelOf(t, router, "monitor-arm")
	if afterReserve.Available != 5 {
		t.Fatalf("available = %d, want 5", afterReserve.Available)
	}

	release := call(t, router, http.MethodPost, "/reservations/release", `{"order_id":"ord_1","reason":"cancelled"}`)
	if release.Code != http.StatusOK {
		t.Fatalf("release status = %d, want 200 (body %s)", release.Code, release.Body)
	}

	afterRelease := levelOf(t, router, "monitor-arm")
	if afterRelease.Available != 8 {
		t.Fatalf("available = %d, want 8", afterRelease.Available)
	}
}

type levelBody struct {
	SKU       string `json:"sku"`
	OnHand    int    `json:"on_hand"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
}

func TestReturnRaisesTheShelf(t *testing.T) {
	router := newRouter(t)

	response := call(t, router, http.MethodPost, "/stock/returns",
		`{"order_id":"ord_1","lines":[{"sku":"monitor-arm","quantity":2}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", response.Code, response.Body)
	}

	level := levelOf(t, router, "monitor-arm")
	if level.OnHand != 10 || level.Available != 10 {
		t.Fatalf("level = %+v, want 10 on hand and available", level)
	}
}

func newRouter(t *testing.T) http.Handler {
	t.Helper()

	stock, err := core.NewService(store.NewMemory(
		core.Level{SKU: "desk-lamp", OnHand: 42},
		core.Level{SKU: "monitor-arm", OnHand: 8},
	))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	router, err := api.NewRouter(stock, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	return router
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

func levelOf(t *testing.T, router http.Handler, sku string) levelBody {
	t.Helper()

	response := call(t, router, http.MethodGet, "/stock/"+sku, "")
	if response.Code != http.StatusOK {
		t.Fatalf("stock status = %d, want 200", response.Code)
	}

	var level levelBody
	if err := json.Unmarshal(response.Body.Bytes(), &level); err != nil {
		t.Fatalf("decode level %q: %v", response.Body, err)
	}
	return level
}
