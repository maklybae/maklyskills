package api

import (
	"errors"
	"log/slog"
	"net/http"

	"orderflow/pkg/httputil"
	"orderflow/services/inventory/internal/core"
)

// fail is where stock errors become status codes; nothing else in the package
// writes an error response.
func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, core.ErrInvalidRequest):
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, core.ErrUnknownSKU):
		httputil.WriteError(w, http.StatusNotFound, "unknown_sku", err.Error())
	case errors.Is(err, core.ErrOutOfStock):
		httputil.WriteError(w, http.StatusConflict, "out_of_stock", err.Error())
	default:
		h.log.ErrorContext(r.Context(), "request failed",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", httputil.RequestIDFrom(r.Context())),
		)
		httputil.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
