package inventory_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/inventory"
)

func TestReserveSendsTheOrderLines(t *testing.T) {
	var (
		gotPath string
		gotBody map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newClient(t, server)
	reservation := core.StockReservation{
		OrderID: "ord_1",
		Items:   []core.Item{{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}},
	}
	if err := client.Reserve(context.Background(), reservation); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	if gotPath != "/reservations" {
		t.Errorf("path = %q, want %q", gotPath, "/reservations")
	}
	if gotBody["order_id"] != "ord_1" {
		t.Errorf("order_id = %v, want ord_1", gotBody["order_id"])
	}
	lines, ok := gotBody["lines"].([]any)
	if !ok || len(lines) != 1 {
		t.Fatalf("lines = %v, want one line", gotBody["lines"])
	}
}

func TestReleaseSendsTheOrderID(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newClient(t, server)
	release := core.StockRelease{OrderID: "ord_1", Reason: "customer changed their mind"}
	if err := client.Release(context.Background(), release); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if gotPath != "/reservations/release" {
		t.Errorf("path = %q, want %q", gotPath, "/reservations/release")
	}
}

func TestResponseMapping(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{name: "accepted", status: http.StatusOK},
		{name: "not enough stock", status: http.StatusConflict, wantErr: core.ErrOutOfStock},
		{name: "rejected request", status: http.StatusBadRequest, wantErr: core.ErrInvalidRequest},
		{name: "inventory is broken", status: http.StatusInternalServerError, wantErr: inventory.ErrUnexpectedResponse},
		{name: "endpoint is gone", status: http.StatusNotFound, wantErr: inventory.ErrUnexpectedResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, `{"error":{"code":"out_of_stock","message":"3 available"}}`)
			}))
			defer server.Close()

			err := newClient(t, server).Reserve(context.Background(), core.StockReservation{OrderID: "ord_1"})
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Reserve() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Reserve() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClientNeedsABaseURL(t *testing.T) {
	if _, err := inventory.NewClient("  "); err == nil {
		t.Fatal("NewClient(\"\") = nil, want an error")
	}
}

func newClient(t *testing.T, server *httptest.Server) *inventory.Client {
	t.Helper()

	client, err := inventory.NewClient(server.URL, inventory.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}
