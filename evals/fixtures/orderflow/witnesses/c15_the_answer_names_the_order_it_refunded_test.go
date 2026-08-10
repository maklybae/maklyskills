//witness:dest services/orders/internal/api
//witness:red refund-bugs,refund-clean

package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The order id in a refund answer is the order the money came off, not the text
// the caller put in the path.
func TestWitnessTheAnswerNamesTheOrderItRefunded(t *testing.T) {
	router, _ := newRouter(t)

	response := call(t, router, http.MethodPost, "/orders/ord_delivered%20/refund",
		`{"key":"sup-1","lines":[{"sku":"desk-lamp","quantity":1}]}`)
	if response.Code != http.StatusOK {
		t.Skip("this build refuses an order id with a space around it")
	}

	var view struct {
		ID      string `json:"id"`
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode body %q: %v", response.Body, err)
	}
	if view.OrderID != "ord_delivered" {
		t.Fatalf("order_id = %q for refund %q, want the order the refund was booked against",
			view.OrderID, view.ID)
	}
}
