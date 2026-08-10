package core

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	"orderflow/pkg/xerrors"
)

// Support quotes the ticket it is working from, which does not fit in the
// length the cancel flow allows for a reason.
const maxRefundReasonLen = 500

// How many times a return is offered to inventory before the refund gives up.
const restockAttempts = 3

// RefundRequest is the input of RefundOrder. Lines are what the customer sends
// back; an empty list refunds everything that is left.
type RefundRequest struct {
	OrderID string
	Key     string
	Reason  string
	Lines   []Item
}

// orderGate serialises the refunds of one order. The map keeps one mutex per
// order this process has refunded, which is a handful of pointers a day.
type orderGate struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newOrderGate() *orderGate {
	return &orderGate{locks: make(map[string]*sync.Mutex)}
}

// enter blocks until the order is free and answers with its release.
func (g *orderGate) enter(orderID string) func() {
	g.mu.Lock()
	lock, held := g.locks[orderID]
	if !held {
		lock = &sync.Mutex{}
		g.locks[orderID] = lock
	}
	g.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}

// RefundOrder gives money back for part or all of an order that was paid for.
//
// The caller's key makes the call safe to repeat: a request that arrives twice
// returns the refund the first one issued instead of paying twice. What is
// refunded goes back on the shelf once the refund is stored.
func (s *Service) RefundOrder(ctx context.Context, req RefundRequest) (Refund, error) {
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Key = strings.TrimSpace(req.Key)
	req.Reason = strings.TrimSpace(req.Reason)
	if err := req.validate(); err != nil {
		return Refund{}, err
	}

	order, err := s.store.Get(ctx, req.OrderID)
	if err != nil {
		return Refund{}, xerrors.Wrapf(err, "load order %s", req.OrderID)
	}
	if issued, ok := refundByKey(order, req.Key); ok {
		return issued, nil
	}
	if err := refundable(order); err != nil {
		return Refund{}, err
	}

	amount, err := refundAmount(order, req.Lines)
	if err != nil {
		return Refund{}, err
	}
	left := linesValue(order.Items)
	if amount > left {
		return Refund{}, xerrors.Wrapf(ErrRefundTooLarge,
			"refund %d of order %s, %d left", amount, order.ID, left)
	}

	issuedAt := s.now().UTC()
	refund := Refund{
		ID:       order.ID + "-r" + strconv.Itoa(len(order.Refunds)+1),
		Key:      req.Key,
		Amount:   amount,
		Reason:   req.Reason,
		Lines:    slices.Clone(req.Lines),
		IssuedAt: issuedAt,
	}

	// Hold the order while it is written so two refunds cannot land on top of
	// each other.
	leave := s.refunds.enter(req.OrderID)
	order.Refunds = append(order.Refunds, refund)
	order.UpdatedAt = issuedAt
	err = s.store.Update(ctx, order)
	leave()
	if err != nil {
		return Refund{}, xerrors.Wrapf(err, "record refund %s", refund.ID)
	}

	if err := s.returnStock(ctx, order, refund); err != nil {
		return refund, err
	}
	return refund, nil
}

// returnStock sends the refunded lines back, giving inventory a couple more
// goes when it is having a bad minute.
func (s *Service) returnStock(ctx context.Context, order Order, refund Refund) error {
	// A refund that names no lines is money only: nothing came back to shelve.
	if len(refund.Lines) == 0 {
		return nil
	}

	ret := StockReturn{OrderID: order.ID, Lines: refund.Lines}

	var err error
	for attempt := 0; attempt < restockAttempts; attempt++ {
		err = s.inventory.Restock(ctx, ret)
		if err == nil {
			return nil
		}
	}
	return xerrors.Wrapf(err, "return the lines of refund %s", refund.ID)
}

func (r RefundRequest) validate() error {
	switch {
	case r.OrderID == "":
		return invalid("id", "order id is required")
	case r.Key == "":
		return invalid("key", "an idempotency key is required")
	case len(r.Reason) > maxRefundReasonLen:
		return invalid("reason", "must be at most "+strconv.Itoa(maxRefundReasonLen)+" characters")
	}

	for _, line := range r.Lines {
		switch {
		case line.SKU == "":
			return invalid("lines", "every line needs a sku")
		case line.Quantity <= 0:
			return invalid("lines", "quantity of "+line.SKU+" must be positive")
		}
	}
	return nil
}

func refundable(order Order) error {
	switch order.Status {
	case StatusPending, StatusPaid, StatusShipped, StatusDelivered:
		return nil
	default:
		return xerrors.Wrapf(ErrNotRefundable, "order %s is %s", order.ID, order.Status)
	}
}

func refundByKey(order Order, key string) (Refund, bool) {
	for i := len(order.Refunds) - 1; i >= 0; i-- {
		if order.Refunds[i].Key == key {
			return order.Refunds[i], true
		}
	}
	return Refund{}, false
}

// refundAmount prices the returned lines.
func refundAmount(order Order, lines []Item) (int64, error) {
	if len(lines) == 0 {
		return linesValue(order.Items) - order.Refunded(), nil
	}

	units, returned := 0, 0
	for _, item := range order.Items {
		units += item.Quantity
	}
	for _, line := range lines {
		item, ok := itemOf(order, line.SKU)
		if !ok {
			return 0, invalid("lines", "sku "+line.SKU+" is not on order "+order.ID)
		}
		if line.Quantity > item.Quantity {
			return 0, invalid("lines", "order "+order.ID+" holds fewer "+line.SKU+" than that")
		}
		returned += line.Quantity
	}
	return linesValue(order.Items) * int64(returned) / int64(units), nil
}

func itemOf(order Order, sku string) (Item, bool) {
	for _, item := range order.Items {
		if item.SKU == sku {
			return item, true
		}
	}
	return Item{}, false
}
