package xerrors_test

import (
	"errors"
	"testing"

	"orderflow/pkg/xerrors"
)

var errRoot = errors.New("connection refused")

func TestWrap(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "single layer",
			err:  xerrors.Wrap(errRoot, "load order"),
			want: "load order: connection refused",
		},
		{
			name: "nested layers read outside in",
			err:  xerrors.Wrap(xerrors.Wrap(errRoot, "load order"), "cancel order"),
			want: "cancel order: load order: connection refused",
		},
		{
			name: "formatted message",
			err:  xerrors.Wrapf(errRoot, "load order %s", "ord_42"),
			want: "load order ord_42: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
			if !errors.Is(tt.err, errRoot) {
				t.Fatalf("errors.Is(%v, errRoot) = false, want true", tt.err)
			}
		})
	}
}

func TestWrapNil(t *testing.T) {
	if err := xerrors.Wrap(nil, "load order"); err != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", err)
	}
	if err := xerrors.Wrapf(nil, "load order %s", "ord_42"); err != nil {
		t.Fatalf("Wrapf(nil) = %v, want nil", err)
	}
}

func TestWrapKeepsTargetType(t *testing.T) {
	type notFound struct{ error }

	target := notFound{errors.New("no such order")}
	err := xerrors.Wrapf(target, "load order %s", "ord_42")

	var got notFound
	if !errors.As(err, &got) {
		t.Fatalf("errors.As did not find notFound in %v", err)
	}
	if got.Error() != "no such order" {
		t.Fatalf("unwrapped error = %q, want %q", got.Error(), "no such order")
	}
}
