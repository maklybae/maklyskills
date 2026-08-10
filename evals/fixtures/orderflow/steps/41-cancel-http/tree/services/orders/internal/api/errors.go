package api

import (
	"errors"
	"log/slog"
	"net/http"

	"orderflow/pkg/httputil"
	"orderflow/services/orders/internal/core"
)

// fail is the only place that turns a domain error into a status code. Anything
// unrecognised is ours, so it is logged here and answered with a 500.
func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, core.ErrInvalidRequest):
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, core.ErrOrderNotFound):
		httputil.WriteError(w, http.StatusNotFound, "order_not_found", err.Error())
	case errors.Is(err, core.ErrOrderExists):
		httputil.WriteError(w, http.StatusConflict, "order_exists", err.Error())
	case errors.Is(err, core.ErrNotCancellable):
		httputil.WriteError(w, http.StatusConflict, "order_not_cancellable", err.Error())
	case errors.Is(err, core.ErrNotShippable):
		httputil.WriteError(w, http.StatusConflict, "order_not_shippable", err.Error())
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
