package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"orderflow/pkg/xerrors"
	"orderflow/services/orders/internal/core"
)

// JSONFile keeps every order in one JSON document. Each call reads and rewrites
// the whole file, which is slow and completely fine at our order volume; when
// it stops being fine the interface is the same for a real database.
type JSONFile struct {
	mu   sync.Mutex
	path string
}

type document struct {
	Orders map[string]core.Order `json:"orders"`
}

// refundBook is the refunds of every order, kept beside the order document.
//
// A build that predates refunds does not know the field they hang off inside an
// order, so it drops them the next time it writes that order out. It has no
// reason to touch this file, which is what makes the money we have given back
// survive a rollback.
type refundBook struct {
	Refunds map[string][]core.Refund `json:"refunds"`
}

// NewJSONFile opens the document at path, creating an empty one if needed.
func NewJSONFile(path string) (*JSONFile, error) {
	if path == "" {
		return nil, errors.New("store: json file path is required")
	}

	file := &JSONFile{path: path}
	orders, err := file.read()
	if err != nil {
		return nil, err
	}
	if err := file.write(orders); err != nil {
		return nil, err
	}
	return file, nil
}

func (f *JSONFile) Create(_ context.Context, order core.Order) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	orders, err := f.read()
	if err != nil {
		return err
	}
	if _, exists := orders[order.ID]; exists {
		return xerrors.Wrapf(core.ErrOrderExists, "create order %s", order.ID)
	}

	orders[order.ID] = order
	return f.write(orders)
}

func (f *JSONFile) Get(_ context.Context, id string) (core.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	orders, err := f.read()
	if err != nil {
		return core.Order{}, err
	}
	order, ok := orders[id]
	if !ok {
		return core.Order{}, xerrors.Wrapf(core.ErrOrderNotFound, "get order %s", id)
	}
	return order, nil
}

func (f *JSONFile) Update(_ context.Context, order core.Order) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	orders, err := f.read()
	if err != nil {
		return err
	}
	if _, exists := orders[order.ID]; !exists {
		return xerrors.Wrapf(core.ErrOrderNotFound, "update order %s", order.ID)
	}

	orders[order.ID] = order
	return f.write(orders)
}

func (f *JSONFile) List(_ context.Context, filter core.Filter) ([]core.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	orders, err := f.read()
	if err != nil {
		return nil, err
	}
	return selectOrders(orders, filter), nil
}

func (f *JSONFile) read() (map[string]core.Order, error) {
	raw, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]core.Order), nil
	}
	if err != nil {
		return nil, xerrors.Wrapf(err, "read %s", f.path)
	}
	if len(raw) == 0 {
		return make(map[string]core.Order), nil
	}

	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, xerrors.Wrapf(err, "decode %s", f.path)
	}
	if doc.Orders == nil {
		doc.Orders = make(map[string]core.Order)
	}

	refunds, err := f.readRefunds()
	if err != nil {
		return nil, err
	}
	for id, recorded := range refunds {
		order, known := doc.Orders[id]
		if !known || len(order.Refunds) > 0 {
			continue
		}
		order.Refunds = recorded
		doc.Orders[id] = order
	}
	return doc.Orders, nil
}

func (f *JSONFile) refundsPath() string {
	return strings.TrimSuffix(f.path, ".json") + ".refunds.json"
}

func (f *JSONFile) readRefunds() (map[string][]core.Refund, error) {
	raw, err := os.ReadFile(f.refundsPath())
	if errors.Is(err, os.ErrNotExist) || len(raw) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, xerrors.Wrapf(err, "read %s", f.refundsPath())
	}

	var book refundBook
	if err := json.Unmarshal(raw, &book); err != nil {
		return nil, xerrors.Wrapf(err, "decode %s", f.refundsPath())
	}
	return book.Refunds, nil
}

func (f *JSONFile) write(orders map[string]core.Order) error {
	if err := writeDocument(f.path, document{Orders: orders}); err != nil {
		return err
	}

	refunds := make(map[string][]core.Refund)
	for id, order := range orders {
		if len(order.Refunds) > 0 {
			refunds[id] = order.Refunds
		}
	}
	if len(refunds) == 0 {
		return nil
	}
	return writeDocument(f.refundsPath(), refundBook{Refunds: refunds})
}

// writeDocument goes through a temporary file in the same directory so a crash
// cannot leave a half written document behind.
func writeDocument(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return xerrors.Wrapf(err, "encode %s", filepath.Base(path))
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".orders-*.json")
	if err != nil {
		return xerrors.Wrapf(err, "create temporary file in %s", dir)
	}
	defer func() {
		_ = os.Remove(temp.Name())
	}()

	if _, err := temp.Write(append(raw, '\n')); err != nil {
		_ = temp.Close()
		return xerrors.Wrapf(err, "write %s", temp.Name())
	}
	if err := temp.Close(); err != nil {
		return xerrors.Wrapf(err, "close %s", temp.Name())
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return xerrors.Wrapf(err, "move %s into place", temp.Name())
	}
	return nil
}
