package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"orderflow/pkg/xerrors"
)

const (
	maxItemsPerOrder = 50
	defaultListLimit = 50
	maxListLimit     = 200
)

// Service is the order flow. Everything the API is allowed to do is a method
// here, and nothing here knows about HTTP.
type Service struct {
	store Store
	ids   IDSource
	now   func() time.Time
}

// IDSource hands out order ids.
type IDSource interface {
	NextID() string
}

// Option overrides a Service default.
type Option func(*Service)

// PlaceRequest is the input of PlaceOrder.
type PlaceRequest struct {
	CustomerID string
	Items      []Item
}

// NewService reports a missing dependency now rather than on the first request.
func NewService(store Store, opts ...Option) (*Service, error) {
	if store == nil {
		return nil, errors.New("orders: store is required")
	}

	svc := &Service{
		store: store,
		ids:   randomIDs{},
		now:   time.Now,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc, nil
}

// WithClock replaces the wall clock, which is how tests get predictable timestamps.
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

// WithIDSource replaces the order id generator.
func WithIDSource(ids IDSource) Option {
	return func(s *Service) {
		if ids != nil {
			s.ids = ids
		}
	}
}

// PlaceOrder validates the request and stores a pending order.
func (s *Service) PlaceOrder(ctx context.Context, req PlaceRequest) (Order, error) {
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	if err := req.validate(); err != nil {
		return Order{}, err
	}

	placed := s.now().UTC()
	order := Order{
		ID:         s.ids.NextID(),
		CustomerID: req.CustomerID,
		Items:      slices.Clone(req.Items),
		Status:     StatusPending,
		PlacedAt:   placed,
		UpdatedAt:  placed,
	}

	if err := s.store.Create(ctx, order); err != nil {
		return Order{}, xerrors.Wrapf(err, "store order %s", order.ID)
	}
	return order, nil
}

// Get returns one order.
func (s *Service) Get(ctx context.Context, id string) (Order, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Order{}, invalid("id", "order id is required")
	}

	order, err := s.store.Get(ctx, id)
	if err != nil {
		return Order{}, xerrors.Wrapf(err, "load order %s", id)
	}
	return order, nil
}

// List returns the orders a filter selects, newest last.
func (s *Service) List(ctx context.Context, filter Filter) ([]Order, error) {
	switch {
	case filter.Limit < 0:
		return nil, invalid("limit", "must not be negative")
	case filter.Limit == 0:
		filter.Limit = defaultListLimit
	case filter.Limit > maxListLimit:
		filter.Limit = maxListLimit
	}

	orders, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, xerrors.Wrap(err, "list orders")
	}
	return orders, nil
}

func (r PlaceRequest) validate() error {
	if r.CustomerID == "" {
		return invalid("customer_id", "is required")
	}
	if len(r.Items) == 0 {
		return invalid("items", "an order needs at least one item")
	}
	if len(r.Items) > maxItemsPerOrder {
		return invalid("items", "an order carries at most "+strconv.Itoa(maxItemsPerOrder)+" items")
	}

	seen := make(map[string]struct{}, len(r.Items))
	for _, item := range r.Items {
		if err := item.validate(); err != nil {
			return err
		}
		if _, duplicate := seen[item.SKU]; duplicate {
			return invalid("items", "sku "+item.SKU+" appears twice, sum the quantities instead")
		}
		seen[item.SKU] = struct{}{}
	}
	return nil
}

type randomIDs struct{}

func (randomIDs) NextID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("orders: no entropy for order ids: " + err.Error())
	}
	return "ord_" + hex.EncodeToString(buf[:])
}
