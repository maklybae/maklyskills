package api

import (
	"net/http"

	"orderflow/pkg/httputil"
	"orderflow/services/inventory/internal/core"
)

type linePayload struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type reservePayload struct {
	OrderID string        `json:"order_id"`
	Lines   []linePayload `json:"lines"`
}

type releasePayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

type restockPayload struct {
	Quantity int `json:"quantity"`
}

type returnPayload struct {
	OrderID string        `json:"order_id"`
	Lines   []linePayload `json:"lines"`
}

type levelView struct {
	SKU       string `json:"sku"`
	OnHand    int    `json:"on_hand"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
}

type levelsView struct {
	Levels []levelView `json:"levels"`
}

type reservationView struct {
	OrderID  string `json:"order_id"`
	Reserved bool   `json:"reserved"`
}

type releaseView struct {
	OrderID  string `json:"order_id"`
	Released bool   `json:"released"`
}

func (h *handler) levels(w http.ResponseWriter, r *http.Request) {
	levels, err := h.stock.Levels(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	views := make([]levelView, 0, len(levels))
	for _, level := range levels {
		views = append(views, viewOf(level))
	}
	httputil.WriteJSON(w, http.StatusOK, levelsView{Levels: views})
}

func (h *handler) level(w http.ResponseWriter, r *http.Request) {
	level, err := h.stock.Level(r.Context(), r.PathValue("sku"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, viewOf(level))
}

func (h *handler) restock(w http.ResponseWriter, r *http.Request) {
	var payload restockPayload
	if err := httputil.Decode(r, &payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	level, err := h.stock.Restock(r.Context(), r.PathValue("sku"), payload.Quantity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, viewOf(level))
}

func (h *handler) reserve(w http.ResponseWriter, r *http.Request) {
	var payload reservePayload
	if err := httputil.Decode(r, &payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	lines := make([]core.Line, 0, len(payload.Lines))
	for _, line := range payload.Lines {
		lines = append(lines, core.Line{SKU: line.SKU, Quantity: line.Quantity})
	}

	if err := h.stock.Reserve(r.Context(), core.ReserveRequest{OrderID: payload.OrderID, Lines: lines}); err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, reservationView{OrderID: payload.OrderID, Reserved: true})
}

func (h *handler) returns(w http.ResponseWriter, r *http.Request) {
	var payload returnPayload
	if err := httputil.Decode(r, &payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	lines := make([]core.Line, 0, len(payload.Lines))
	for _, line := range payload.Lines {
		lines = append(lines, core.Line{SKU: line.SKU, Quantity: line.Quantity})
	}

	levels, err := h.stock.Return(r.Context(), core.ReturnRequest{OrderID: payload.OrderID, Lines: lines})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	views := make([]levelView, 0, len(levels))
	for _, level := range levels {
		views = append(views, viewOf(level))
	}
	httputil.WriteJSON(w, http.StatusOK, levelsView{Levels: views})
}

func (h *handler) release(w http.ResponseWriter, r *http.Request) {
	var payload releasePayload
	if err := httputil.Decode(r, &payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	released, err := h.stock.Release(r.Context(), payload.OrderID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, releaseView{OrderID: payload.OrderID, Released: released})
}

func viewOf(level core.Level) levelView {
	return levelView{
		SKU:       level.SKU,
		OnHand:    level.OnHand,
		Reserved:  level.Reserved,
		Available: level.Available(),
	}
}
