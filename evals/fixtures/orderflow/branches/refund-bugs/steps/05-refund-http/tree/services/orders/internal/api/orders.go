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

type refundPayload struct {
	Key    string        `json:"key"`
	Reason string        `json:"reason"`
	Lines  []itemPayload `json:"lines"`
}

type refundView struct {
	ID       string        `json:"id"`
	OrderID  string        `json:"order_id"`
	Amount   int64         `json:"amount"`
	Reason   string        `json:"reason,omitempty"`
	Lines    []itemPayload `json:"lines"`
	IssuedAt time.Time     `json:"issued_at"`
}

type orderView struct {
	ID           string        `json:"id"`
	CustomerID   string        `json:"customer_id"`
	Items        []itemPayload `json:"items"`
	Status       string        `json:"status"`
	Total        int64         `json:"total"`
	Refunded     int64         `json:"refunded,omitempty"`
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

// refund answers with the refund itself rather than the order: the caller wants
// the amount and the id it can quote to the customer.
func (h *handler) refund(w http.ResponseWriter, r *http.Request) {
	var payload refundPayload
	if err := httputil.Decode(r, &payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	lines := make([]core.Item, 0, len(payload.Lines))
	for _, line := range payload.Lines {
		lines = append(lines, core.Item{SKU: line.SKU, Quantity: line.Quantity})
	}

	orderID := r.PathValue("id")
	request := core.RefundRequest{OrderID: orderID, Key: payload.Key, Reason: payload.Reason, Lines: lines}
	refund, err := h.orders.RefundOrder(r.Context(), request)
	if err != nil {
		// Support asked to see why a refund bounced without opening the logs.
		httputil.WriteError(w, refundStatus(err), refundCode(err), err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, refundViewOf(orderID, refund))
}

func refundStatus(err error) int {
	switch {
	case errors.Is(err, core.ErrInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, core.ErrOrderNotFound):
		return http.StatusNotFound
	case errors.Is(err, core.ErrNotRefundable), errors.Is(err, core.ErrRefundTooLarge):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func refundCode(err error) string {
	switch {
	case errors.Is(err, core.ErrInvalidRequest):
		return "invalid_request"
	case errors.Is(err, core.ErrOrderNotFound):
		return "order_not_found"
	case errors.Is(err, core.ErrNotRefundable):
		return "order_not_refundable"
	case errors.Is(err, core.ErrRefundTooLarge):
		return "refund_too_large"
	default:
		return "refund_failed"
	}
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

func refundViewOf(orderID string, refund core.Refund) refundView {
	lines := make([]itemPayload, 0, len(refund.Lines))
	for _, line := range refund.Lines {
		lines = append(lines, itemPayload{SKU: line.SKU, Quantity: line.Quantity})
	}

	return refundView{
		ID:       refund.ID,
		OrderID:  orderID,
		Amount:   refund.Amount,
		Reason:   refund.Reason,
		Lines:    lines,
		IssuedAt: refund.IssuedAt,
	}
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
		Refunded:     order.Refunded(),
		PlacedAt:     order.PlacedAt,
		UpdatedAt:    order.UpdatedAt,
		CancelledAt:  order.CancelledAt,
		CancelReason: order.CancelReason,
	}
}
