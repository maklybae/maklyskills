package httputil_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orderflow/pkg/httputil"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		inbound  string
		wantSame bool
	}{
		{name: "generated when absent", inbound: "", wantSame: false},
		{name: "reused when present", inbound: "req-from-caller", wantSame: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			handler := httputil.Chain(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					seen = httputil.RequestIDFrom(r.Context())
				}),
				httputil.RequestID(),
			)

			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			if tt.inbound != "" {
				req.Header.Set(httputil.HeaderRequestID, tt.inbound)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if seen == "" {
				t.Fatal("handler saw no request id in the context")
			}
			if got := rec.Header().Get(httputil.HeaderRequestID); got != seen {
				t.Fatalf("response header = %q, context = %q", got, seen)
			}
			if same := seen == tt.inbound; same != tt.wantSame {
				t.Fatalf("id reused = %v, want %v (id %q)", same, tt.wantSame, seen)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	handler := httputil.Chain(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("store went away")
		}),
		httputil.Recover(discardLogger()),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/orders", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	var body httputil.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != "internal" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "internal")
	}
}

func TestDecode(t *testing.T) {
	type payload struct {
		Reason string `json:"reason"`
	}

	tests := []struct {
		name    string
		body    string
		want    payload
		wantErr error
	}{
		{name: "valid", body: `{"reason":"duplicate order"}`, want: payload{Reason: "duplicate order"}},
		{name: "empty body", body: "", wantErr: httputil.ErrEmptyBody},
		{name: "unknown field", body: `{"resaon":"typo"}`, wantErr: errDecode},
		{name: "trailing value", body: `{"reason":"one"}{"reason":"two"}`, wantErr: errDecode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))

			var got payload
			err := httputil.Decode(req, &got)
			switch {
			case tt.wantErr == nil && err != nil:
				t.Fatalf("Decode() = %v, want nil", err)
			case tt.wantErr == errDecode && err == nil:
				t.Fatal("Decode() = nil, want a decoding error")
			case tt.wantErr != nil && tt.wantErr != errDecode && !errors.Is(err, tt.wantErr):
				t.Fatalf("Decode() = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got != tt.want {
				t.Fatalf("decoded = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// errDecode marks the cases where any decoding error is an acceptable answer.
var errDecode = errors.New("decoding error")

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
