package core

import (
	"encoding/json"
	"strconv"

	"orderflow/pkg/xerrors"
)

// Status is a point in the order lifecycle.
type Status uint8

// Order lifecycle.
const (
	StatusPending Status = iota
	StatusPaid
	StatusShipped
	StatusDelivered
)

var statusNames = map[Status]string{
	StatusPending:   "pending",
	StatusPaid:      "paid",
	StatusShipped:   "shipped",
	StatusDelivered: "delivered",
}

// ParseStatus resolves the wire name of a status.
func ParseStatus(name string) (Status, bool) {
	for status, wire := range statusNames {
		if wire == name {
			return status, true
		}
	}
	return 0, false
}

func (s Status) String() string {
	name, ok := statusNames[s]
	if !ok {
		return "status(" + strconv.Itoa(int(s)) + ")"
	}
	return name
}

func (s Status) MarshalJSON() ([]byte, error) {
	name, ok := statusNames[s]
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
