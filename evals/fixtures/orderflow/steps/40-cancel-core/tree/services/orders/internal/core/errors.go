package core

import "errors"

// Everything the API turns into a status code starts here.
var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrOrderNotFound  = errors.New("order not found")
	ErrOrderExists    = errors.New("order already exists")
	ErrNotCancellable = errors.New("order cannot be cancelled")
	ErrNotShippable   = errors.New("order cannot be shipped")
	ErrOutOfStock     = errors.New("insufficient stock")
)

// ValidationError names the field that made a request invalid.
type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Reason
}

// Is lets callers match any validation failure with ErrInvalidRequest.
func (e ValidationError) Is(target error) bool {
	return target == ErrInvalidRequest
}

func invalid(field, reason string) error {
	return ValidationError{Field: field, Reason: reason}
}
