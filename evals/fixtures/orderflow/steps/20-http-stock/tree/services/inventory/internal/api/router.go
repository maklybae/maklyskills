package api

import (
	"errors"
	"log/slog"
	"net/http"

	"orderflow/pkg/httputil"
	"orderflow/services/inventory/internal/core"
)

type handler struct {
	stock *core.Service
	log   *slog.Logger
}

// NewRouter wires the HTTP surface of the inventory service.
func NewRouter(stock *core.Service, log *slog.Logger) (http.Handler, error) {
	if stock == nil {
		return nil, errors.New("api: stock service is required")
	}
	if log == nil {
		return nil, errors.New("api: logger is required")
	}

	h := &handler{stock: stock, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /stock", h.levels)
	mux.HandleFunc("GET /stock/{sku}", h.level)
	mux.HandleFunc("POST /stock/{sku}/restock", h.restock)
	mux.HandleFunc("POST /reservations", h.reserve)
	mux.HandleFunc("POST /reservations/release", h.release)

	return httputil.Chain(mux,
		httputil.RequestID(),
		httputil.Logger(log),
		httputil.Recover(log),
	), nil
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
