package core

import (
	"context"
	"hash/fnv"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"orderflow/pkg/xerrors"
)

// Support quotes the ticket it is working from, which does not fit in the
// length the cancel flow allows for a reason.
const maxRefundReasonLen = 500

// An idempotency key is a handle the caller chose, not somewhere to put data.
const maxRefundKeyLen = 64

// How long the stock side of a committed refund may take.
const returnDeadline = 10 * time.Second

// RefundRequest is the input of RefundOrder. Lines are what the customer sends
// back; an empty list refunds everything that is left.
type RefundRequest struct {
	OrderID string
	Key     string
	Reason  string
	Lines   []Item
}

// orderGate serialises the refunds of one order over a fixed set of locks. Two
// orders that land on the same lock wait for each other, which costs a moment;
// an id we have never seen costs nothing at all.
type orderGate struct {
	locks [64]sync.Mutex
}

// enter blocks until the order is free and answers with its release.
func (g *orderGate) enter(orderID string) func() {
	digest := fnv.New32a()
	_, _ = digest.Write([]byte(orderID))

	lock := &g.locks[int(digest.Sum32())%len(g.locks)]
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

	// One refund of an order at a time: the key, the ceiling and the write have
	// to be decided on the same version of it.
	leave := s.refunds.enter(req.OrderID)
	defer leave()

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
	if amount == 0 && len(req.Lines) == 0 {
		return Refund{}, invalid("lines", "order "+order.ID+" has nothing left to refund")
	}
	left := linesValue(order.Items) - order.Refunded()
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

	order.Refunds = append(order.Refunds, refund)
	order.UpdatedAt = issuedAt
	if err := s.store.Update(ctx, order); err != nil {
		return Refund{}, xerrors.Wrapf(err, "record refund %s", refund.ID)
	}

	if err := s.returnStock(ctx, order, refund); err != nil {
		return refund, err
	}
	return refund, nil
}

// returnStock settles the stock side of a refund that is already recorded.
//
// It runs on a context of its own: the money has gone back, and a caller that
// hung up in the meantime must not leave the shelf saying something else. The
// return is sent once - inventory counts returns rather than keying them, so a
// retry we are not sure about would shelve the same items twice, and a return
// that never arrived is picked up by the warehouse count instead.
func (s *Service) returnStock(ctx context.Context, order Order, refund Refund) error {
	// A refund that names no lines is money only: nothing came back to shelve.
	if len(refund.Lines) == 0 {
		return nil
	}

	// Units go back on the shelf only if they ever left it. What the warehouse
	// still holds stays held: the hold is what the cancel path gives back.
	if order.Status != StatusShipped && order.Status != StatusDelivered {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), returnDeadline)
	defer cancel()

	ret := StockReturn{OrderID: order.ID, Lines: refund.Lines}
	if err := s.inventory.Restock(ctx, ret); err != nil {
		return xerrors.Wrapf(err, "return the lines of refund %s", refund.ID)
	}
	return nil
}

func (r RefundRequest) validate() error {
	switch {
	case r.OrderID == "":
		return invalid("id", "order id is required")
	case r.Key == "":
		return invalid("key", "an idempotency key is required")
	case len(r.Key) > maxRefundKeyLen:
		return invalid("key", "must be at most "+strconv.Itoa(maxRefundKeyLen)+" characters")
	case len(r.Reason) > maxRefundReasonLen:
		return invalid("reason", "must be at most "+strconv.Itoa(maxRefundReasonLen)+" characters")
	}

	seen := make(map[string]struct{}, len(r.Lines))
	for _, line := range r.Lines {
		switch {
		case line.SKU == "":
			return invalid("lines", "every line needs a sku")
		case line.Quantity <= 0:
			return invalid("lines", "quantity of "+line.SKU+" must be positive")
		}
		if _, duplicate := seen[line.SKU]; duplicate {
			return invalid("lines", "sku "+line.SKU+" is on two lines, add the quantities up")
		}
		seen[line.SKU] = struct{}{}
	}
	return nil
}

func refundable(order Order) error {
	switch order.Status {
	case StatusPaid, StatusShipped, StatusDelivered:
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

// refundAmount prices the returned lines against what the order still has to
// give back.
func refundAmount(order Order, lines []Item) (int64, error) {
	if len(lines) == 0 {
		return linesValue(order.Items) - order.Refunded(), nil
	}

	returned := returnedUnits(order)

	var amount int64
	for _, line := range lines {
		item, ok := itemOf(order, line.SKU)
		if !ok {
			return 0, invalid("lines", "sku "+line.SKU+" is not on order "+order.ID)
		}
		if line.Quantity > item.Quantity-returned[line.SKU] {
			return 0, invalid("lines", "order "+order.ID+" has fewer "+line.SKU+" left to give back")
		}
		amount += item.UnitPrice * int64(line.Quantity)
	}
	return amount, nil
}

// returnedUnits counts what earlier refunds of this order already sent back.
func returnedUnits(order Order) map[string]int {
	returned := make(map[string]int)
	for _, refund := range order.Refunds {
		for _, line := range refund.Lines {
			returned[line.SKU] += line.Quantity
		}
	}
	return returned
}

func itemOf(order Order, sku string) (Item, bool) {
	for _, item := range order.Items {
		if item.SKU == sku {
			return item, true
		}
	}
	return Item{}, false
}
