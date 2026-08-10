package core

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"orderflow/pkg/xerrors"
)

const maxLinesPerReservation = 50

// Service is the stock flow: reservations, releases and what is on the shelf.
type Service struct {
	store Store
}

// ReserveRequest holds stock for one order.
type ReserveRequest struct {
	OrderID string
	Lines   []Line
}

// NewService needs a store; without one there is nothing to count.
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, errors.New("inventory: store is required")
	}
	return &Service{store: store}, nil
}

// Reserve takes all the lines of an order or none of them. A second reserve
// under an order id that already holds stock is a no-op, so the caller can
// retry a request that timed out.
func (s *Service) Reserve(ctx context.Context, req ReserveRequest) error {
	req.OrderID = strings.TrimSpace(req.OrderID)
	if err := req.validate(); err != nil {
		return err
	}

	if err := s.store.Reserve(ctx, req.OrderID, req.Lines); err != nil {
		return xerrors.Wrapf(err, "hold stock for order %s", req.OrderID)
	}
	return nil
}

// Release gives the stock of an order back and reports whether anything was
// held. An unknown order id is not an error: the caller repeats releases.
func (s *Service) Release(ctx context.Context, orderID string) (bool, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return false, invalid("order_id", "is required")
	}

	released, err := s.store.Release(ctx, orderID)
	if err != nil {
		return false, xerrors.Wrapf(err, "give the stock of order %s back", orderID)
	}
	return released, nil
}

// Level returns the stock of one sku.
func (s *Service) Level(ctx context.Context, sku string) (Level, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return Level{}, invalid("sku", "is required")
	}

	level, err := s.store.Level(ctx, sku)
	if err != nil {
		return Level{}, xerrors.Wrapf(err, "load stock of %s", sku)
	}
	return level, nil
}

// Levels returns the whole shelf, sku by sku.
func (s *Service) Levels(ctx context.Context) ([]Level, error) {
	levels, err := s.store.Levels(ctx)
	if err != nil {
		return nil, xerrors.Wrap(err, "list stock")
	}
	return levels, nil
}

// Restock books goods in. An unknown sku starts a new line on the shelf.
func (s *Service) Restock(ctx context.Context, sku string, quantity int) (Level, error) {
	sku = strings.TrimSpace(sku)
	switch {
	case sku == "":
		return Level{}, invalid("sku", "is required")
	case quantity <= 0:
		return Level{}, invalid("quantity", "must be positive")
	}

	level, err := s.store.Restock(ctx, sku, quantity)
	if err != nil {
		return Level{}, xerrors.Wrapf(err, "restock %s", sku)
	}
	return level, nil
}

func (r ReserveRequest) validate() error {
	if r.OrderID == "" {
		return invalid("order_id", "is required")
	}
	if len(r.Lines) == 0 {
		return invalid("lines", "a reservation needs at least one line")
	}
	if len(r.Lines) > maxLinesPerReservation {
		return invalid("lines", "at most "+strconv.Itoa(maxLinesPerReservation)+" lines per reservation")
	}

	seen := make(map[string]struct{}, len(r.Lines))
	for _, line := range r.Lines {
		switch {
		case strings.TrimSpace(line.SKU) == "":
			return invalid("lines", "every line needs a sku")
		case line.Quantity <= 0:
			return invalid("lines", "quantity of "+line.SKU+" must be positive")
		}
		if _, duplicate := seen[line.SKU]; duplicate {
			return invalid("lines", "sku "+line.SKU+" is on two lines, merge them")
		}
		seen[line.SKU] = struct{}{}
	}
	return nil
}
