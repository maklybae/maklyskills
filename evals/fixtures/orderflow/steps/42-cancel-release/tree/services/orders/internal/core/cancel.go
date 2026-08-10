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

// CancelOrder cancels an order and gives the stock it holds back.
//
// Cancelling twice is safe: the second call leaves the stored order untouched
// and only repeats the stock release, which inventory keys by order id. An
// order that already left the warehouse comes back as ErrNotCancellable.
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
		return order, s.releaseStock(ctx, order)
	}
	if err := cancellable(order); err != nil {
		return Order{}, err
	}

	cancelled := s.now().UTC()
	order.Status = StatusCancelled
	order.CancelReason = req.Reason
	order.CancelledAt = &cancelled
	order.UpdatedAt = cancelled

	// The status is written before the release: a release we missed is repeated
	// by the next cancel, a status we missed leaves the customer looking at an
	// order that is still alive.
	if err := s.store.Update(ctx, order); err != nil {
		return Order{}, xerrors.Wrapf(err, "cancel order %s", order.ID)
	}
	if err := s.releaseStock(ctx, order); err != nil {
		return order, err
	}
	return order, nil
}

func (s *Service) releaseStock(ctx context.Context, order Order) error {
	release := StockRelease{OrderID: order.ID, Reason: order.CancelReason}
	if err := s.inventory.Release(ctx, release); err != nil {
		return xerrors.Wrapf(err, "release stock of order %s", order.ID)
	}
	return nil
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
