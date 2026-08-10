package core

import (
	"encoding/json"
	"strconv"

	"orderflow/pkg/xerrors"
	"orderflow/services/orders/internal/generated"
)

// Status is a point in the order lifecycle.
type Status uint8

// Order lifecycle. The wire names are generated from this block, so a new state
// needs a `make generate` in the same change.
const (
	StatusPending Status = iota
	StatusPaid
	StatusShipped
	StatusDelivered
)

// ParseStatus resolves the wire name of a status.
func ParseStatus(name string) (Status, bool) {
	code, ok := generated.StatusCode(name)
	if !ok {
		return 0, false
	}
	return Status(code), true
}

func (s Status) String() string {
	name, ok := generated.StatusName(uint8(s))
	if !ok {
		return "status(" + strconv.Itoa(int(s)) + ")"
	}
	return name
}

func (s Status) MarshalJSON() ([]byte, error) {
	name, ok := generated.StatusName(uint8(s))
	if !ok {
		return nil, xerrors.Wrapf(ErrInvalidRequest, "order status %d has no name", uint8(s))
	}
	return json.Marshal(name)
}

func (s *Status) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return xerrors.Wrap(err, "decode order status")
	}
	parsed, ok := ParseStatus(name)
	if !ok {
		return xerrors.Wrapf(ErrInvalidRequest, "unknown order status %q", name)
	}
	*s = parsed
	return nil
}
