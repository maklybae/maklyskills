//witness:dest services/orders/internal/inventory
//witness:red refund-bugs

package inventory_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"orderflow/pkg/httputil"
	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/inventory"
)

// The return is a call like the other two: it goes out through the client the
// service was given, so it carries the request id and is bounded by the timeout
// that client was configured with.
func TestWitnessReturnGoesThroughTheClient(t *testing.T) {
	var gotID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = r.Header.Get(httputil.HeaderRequestID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ret := core.StockReturn{OrderID: "ord_1", Lines: []core.Item{{SKU: "desk-lamp", Quantity: 1}}}
	if err := newClient(t, server).Restock(witnessRequestContext(t, "req-42"), ret); err != nil {
		t.Fatalf("Restock() error = %v", err)
	}
	if gotID != "req-42" {
		t.Errorf("request id = %q, want the callers id carried across the hop", gotID)
	}

	quiet := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-quiet
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(quiet)
		slow.Close()
	}()

	impatient, err := inventory.NewClient(slow.URL, inventory.WithHTTPClient(&http.Client{Timeout: 100 * time.Millisecond}))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- impatient.Restock(context.Background(), ret)
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Restock() = nil, want the timeout back")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Restock() ignored the timeout of the client it was given")
	}
}

func witnessRequestContext(t *testing.T, id string) context.Context {
	t.Helper()

	var ctx context.Context
	handler := httputil.Chain(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { ctx = r.Context() }),
		httputil.RequestID(),
	)

	request := httptest.NewRequest(http.MethodPost, "/orders/ord_1/refund", nil)
	request.Header.Set(httputil.HeaderRequestID, id)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	return ctx
}
