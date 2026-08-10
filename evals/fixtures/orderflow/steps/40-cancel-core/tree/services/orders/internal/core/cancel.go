package core

import (
	"context"
	"strconv"
	"strings"

	"orderflow/pkg/xerrors"
)

const maxCancelReasonLen = 240

// CancelRequest is the input of CancelOrder.
type CancelRequest struct {
	OrderID string
	Reason  string
}

// CancelOrder cancels an order that has not left the warehouse yet.
//
// Cancelling twice is safe: the second call finds the order cancelled and
// leaves it alone, so support can retry a request that timed out.
func (s *Service) CancelOrder(ctx context.Context, req CancelRequest) (Order, error) {
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Reason = strings.TrimSpace(req.Reason)
	if err := req.validate(); err != nil {
		return Order{}, err
	}

	order, err := s.store.Get(ctx, req.OrderID)
	if err != nil {
		return Order{}, xerrors.Wrapf(err, "load order %s", req.OrderID)
	}
	if order.Status == StatusCancelled {
		return order, nil
	}
	if err := cancellable(order); err != nil {
		return Order{}, err
	}

	cancelled := s.now().UTC()
	order.Status = StatusCancelled
	order.CancelReason = req.Reason
	order.CancelledAt = &cancelled
	order.UpdatedAt = cancelled

	if err := s.store.Update(ctx, order); err != nil {
		return Order{}, xerrors.Wrapf(err, "cancel order %s", order.ID)
	}
	return order, nil
}

func (r CancelRequest) validate() error {
	switch {
	case r.OrderID == "":
		return invalid("id", "order id is required")
	case len(r.Reason) > maxCancelReasonLen:
		return invalid("reason", "must be at most "+strconv.Itoa(maxCancelReasonLen)+" characters")
	}
	return nil
}

func cancellable(order Order) error {
	switch order.Status {
	case StatusPending, StatusPaid:
		return nil
	default:
		return xerrors.Wrapf(ErrNotCancellable, "order %s is %s", order.ID, order.Status)
	}
}
