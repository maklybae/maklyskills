#!/bin/sh
#witness:red refund-clean
# Witness for the test that names the serialisation of refunds: take the gate
# away and run that test on one processor, where the scheduler alone would
# carry it. A test that pins the gate fails; one that pins the scheduler passes.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
checkout=$(cd "$1" && pwd)
target=services/orders/internal/core/refund.go

# A build with no test of its own for the gate has nothing to weaken.
grep -q 'func TestRefundOrderIsSerialisedPerOrder' \
	"$checkout/services/orders/internal/core/refund_test.go" || exit 0

work=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-witness.XXXXXX")
trap 'rm -rf "$work"' EXIT INT TERM
tar -cf - -C "$checkout" --exclude .git . | tar -xf - -C "$work"

awk '
/^func \(g \*orderGate\) enter\(orderID string\) func\(\) \{$/ {
	print
	print "\t_, _ = fnv.New32a().Write([]byte(orderID))"
	print "\treturn func() {}"
	inside = 1
	next
}
inside && /^\}$/ { inside = 0; print; next }
inside { next }
{ print }
' "$checkout/$target" >"$work/$target"

if cmp -s "$checkout/$target" "$work/$target"; then
	echo "c22: the gate is not where this witness looks for it" >&2
	exit 2
fi

if (cd "$work" && GOMAXPROCS=1 ORDERFLOW_TEST_MODE=memory go test \
	./services/orders/internal/core/ -race -count=1 \
	-run '^TestRefundOrderIsSerialisedPerOrder$' >"$work/test.log" 2>&1); then
	echo "c22: the test named for the gate passes without one"
	exit 1
fi
echo "c22: the test named for the gate caught its removal"
exit 0
