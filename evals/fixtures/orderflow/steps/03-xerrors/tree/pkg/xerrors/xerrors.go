// Package xerrors adds context to errors without losing the value underneath.
package xerrors

import "fmt"

// Wrap returns err described by msg, or nil when err is nil.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &wrapped{msg: msg, err: err}
}

// Wrapf is Wrap with a formatted message.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &wrapped{msg: fmt.Sprintf(format, args...), err: err}
}

type wrapped struct {
	msg string
	err error
}

func (w *wrapped) Error() string {
	return w.msg + ": " + w.err.Error()
}

func (w *wrapped) Unwrap() error {
	return w.err
}
