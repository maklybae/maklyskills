//witness:dest services/orders/internal/api
//witness:red refund-bugs

package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"orderflow/pkg/httputil"
)

// A failure the client cannot do anything about answers with the same envelope
// every other endpoint uses, and says nothing about what broke inside.
func TestWitnessRefundFailureKeepsInternalsQuiet(t *testing.T) {
	router, orders := newRouter(t)
	orders.FailUpdate(errors.New("ydb: session pool exhausted at 10.2.3.4:2135"))

	response := call(t, router, http.MethodPost, "/orders/ord_delivered/refund",
		`{"key":"sup-1","lines":[{"sku":"desk-lamp","quantity":1}]}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", response.Code, response.Body)
	}

	var failure httputil.ErrorBody
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
		t.Fatalf("decode error body %q: %v", response.Body, err)
	}
	if strings.Contains(failure.Error.Message, "session pool") ||
		strings.Contains(failure.Error.Message, "10.2.3.4") {
		t.Fatalf("the response repeats what broke inside: %q", failure.Error.Message)
	}
	if failure.Error.Message != "internal error" {
		t.Fatalf("message = %q, want the same envelope the other handlers use", failure.Error.Message)
	}
}
