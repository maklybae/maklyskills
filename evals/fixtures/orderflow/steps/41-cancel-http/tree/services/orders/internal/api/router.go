package api

import (
	"errors"
	"log/slog"
	"net/http"

	"orderflow/pkg/httputil"
	"orderflow/services/orders/internal/core"
)

type handler struct {
	orders *core.Service
	log    *slog.Logger
}

// NewRouter wires the HTTP surface of the orders service.
func NewRouter(orders *core.Service, log *slog.Logger) (http.Handler, error) {
	if orders == nil {
		return nil, errors.New("api: order service is required")
	}
	if log == nil {
		return nil, errors.New("api: logger is required")
	}

	h := &handler{orders: orders, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /orders", h.place)
	mux.HandleFunc("GET /orders", h.list)
	mux.HandleFunc("GET /orders/{id}", h.get)
	mux.HandleFunc("POST /orders/{id}/pay", h.pay)
	mux.HandleFunc("POST /orders/{id}/ship", h.ship)
	mux.HandleFunc("POST /orders/{id}/cancel", h.cancel)

	// Recover sits innermost so a panicking request still leaves an access log line.
	return httputil.Chain(mux,
		httputil.RequestID(),
		httputil.Logger(log),
		httputil.Recover(log),
	), nil
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
