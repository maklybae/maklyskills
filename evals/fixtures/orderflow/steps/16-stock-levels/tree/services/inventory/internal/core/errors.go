package core

import "errors"

// The failures a caller is allowed to see. Anything else is a 500.
var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrUnknownSKU     = errors.New("unknown sku")
	ErrOutOfStock     = errors.New("not enough stock")
)

// ValidationError says which field of the request is wrong and why.
type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Reason
}

// Is makes every field failure match ErrInvalidRequest.
func (e ValidationError) Is(target error) bool {
	return target == ErrInvalidRequest
}

func invalid(field, reason string) error {
	return ValidationError{Field: field, Reason: reason}
}
