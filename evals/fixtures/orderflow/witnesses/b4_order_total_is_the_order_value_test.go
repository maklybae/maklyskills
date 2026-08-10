//witness:dest services/orders/internal/api
//witness:red refund-bugs

package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The total of an order is what was ordered. What has gone back is reported
// separately, so a partial refund cannot quietly restate the order value.
func TestWitnessOrderTotalIsTheOrderValue(t *testing.T) {
	router, _ := newRouter(t)

	refund := call(t, router, http.MethodPost, "/orders/ord_delivered/refund",
		`{"key":"sup-1","lines":[{"sku":"desk-lamp","quantity":1}]}`)
	if refund.Code != http.StatusOK {
		t.Fatalf("refund status = %d, want 200 (body %s)", refund.Code, refund.Body)
	}

	response := call(t, router, http.MethodGet, "/orders/ord_delivered", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	var view struct {
		Total    int64 `json:"total"`
		Refunded int64 `json:"refunded"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode body %q: %v", response.Body, err)
	}
	if view.Total != 4900 {
		t.Fatalf("total = %d after a refund, want the order value 4900", view.Total)
	}
}
