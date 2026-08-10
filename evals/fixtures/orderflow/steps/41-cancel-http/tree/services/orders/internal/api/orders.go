package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"orderflow/pkg/httputil"
	"orderflow/services/orders/internal/core"
)

type itemPayload struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

type placePayload struct {
	CustomerID string        `json:"customer_id"`
	Items      []itemPayload `json:"items"`
}

type cancelPayload struct {
	Reason string `json:"reason"`
}

type orderView struct {
	ID           string        `json:"id"`
	CustomerID   string        `json:"customer_id"`
	Items        []itemPayload `json:"items"`
	Status       string        `json:"status"`
	Total        int64         `json:"total"`
	PlacedAt     time.Time     `json:"placed_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	CancelledAt  *time.Time    `json:"cancelled_at,omitempty"`
	CancelReason string        `json:"cancel_reason,omitempty"`
}

type listView struct {
	Orders []orderView `json:"orders"`
}

func (h *handler) place(w http.ResponseWriter, r *http.Request) {
	var payload placePayload
	if err := httputil.Decode(r, &payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	items := make([]core.Item, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, core.Item{SKU: item.SKU, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}

	order, err := h.orders.PlaceOrder(r.Context(), core.PlaceRequest{CustomerID: payload.CustomerID, Items: items})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, viewOf(order))
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	order, err := h.orders.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, viewOf(order))
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	orders, err := h.orders.List(r.Context(), filter)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	views := make([]orderView, 0, len(orders))
	for _, order := range orders {
		views = append(views, viewOf(order))
	}
	httputil.WriteJSON(w, http.StatusOK, listView{Orders: views})
}

func (h *handler) pay(w http.ResponseWriter, r *http.Request) {
	order, err := h.orders.MarkPaid(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, viewOf(order))
}

func (h *handler) ship(w http.ResponseWriter, r *http.Request) {
	order, err := h.orders.MarkShipped(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, viewOf(order))
}

// cancel accepts a request without a body: a reason is welcome, not required.
func (h *handler) cancel(w http.ResponseWriter, r *http.Request) {
	var payload cancelPayload
	if err := httputil.Decode(r, &payload); err != nil && !errors.Is(err, httputil.ErrEmptyBody) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	request := core.CancelRequest{OrderID: r.PathValue("id"), Reason: payload.Reason}
	order, err := h.orders.CancelOrder(r.Context(), request)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, viewOf(order))
}

func filterFromQuery(r *http.Request) (core.Filter, error) {
	query := r.URL.Query()
	filter := core.Filter{CustomerID: query.Get("customer_id")}

	for _, name := range query["status"] {
		status, ok := core.ParseStatus(name)
		if !ok {
			return core.Filter{}, core.ValidationError{Field: "status", Reason: "unknown status " + name}
		}
		filter.Statuses = append(filter.Statuses, status)
	}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return core.Filter{}, core.ValidationError{Field: "limit", Reason: "must be a number"}
		}
		filter.Limit = limit
	}
	return filter, nil
}

func viewOf(order core.Order) orderView {
	items := make([]itemPayload, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, itemPayload{SKU: item.SKU, Quantity: item.Quantity, UnitPrice: item.UnitPrice})
	}

	return orderView{
		ID:           order.ID,
		CustomerID:   order.CustomerID,
		Items:        items,
		Status:       order.Status.String(),
		Total:        order.Total(),
		PlacedAt:     order.PlacedAt,
		UpdatedAt:    order.UpdatedAt,
		CancelledAt:  order.CancelledAt,
		CancelReason: order.CancelReason,
	}
}
