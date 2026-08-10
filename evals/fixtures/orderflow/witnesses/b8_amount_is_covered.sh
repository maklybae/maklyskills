#!/bin/sh
#witness:red refund-bugs
# Witness for the refund amount test: change what the amount computation
# returns and the suite has to notice.
#
# Usage: b8_amount_is_covered.sh <checkout>
# Exit 0  the suite caught the change  (the amount is covered)
# Exit 1  the suite stayed green       (the amount is asserted against itself)
set -eu

[ $# -eq 1 ] || {
	echo "usage: $(basename "$0") <checkout>" >&2
	exit 2
}
checkout=$(cd "$1" && pwd)
refund=services/orders/internal/core/refund.go

[ -f "$checkout/$refund" ] || {
	echo "b8: $refund is not in $checkout" >&2
	exit 2
}

work=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-witness-b8.XXXXXX")
cleanup() {
	rm -rf "$work"
}
trap cleanup EXIT INT TERM

tar -cf - -C "$checkout" --exclude .git . | tar -xf - -C "$work"

# Inside refundAmount, make every value it returns a zero. The validation paths
# (`return 0, invalid(...)`) are left alone, so only the amount changes.
awk '
	/^func refundAmount\(/ { inside = 1 }
	inside && /^\t+return .*, nil$/ { sub(/return .*, nil$/, "return 0, nil") }
	inside && /^\}$/ { inside = 0 }
	{ print }
' "$checkout/$refund" >"$work/$refund"

if ! cmp -s "$checkout/$refund" "$work/$refund"; then
	:
else
	echo "b8: the amount computation was not changed, the witness cannot judge" >&2
	exit 2
fi

if (cd "$work" && make test >"$work/test.log" 2>&1); then
	echo "b8: the suite is green with a refund amount of zero"
	exit 1
fi

echo "b8: the suite caught the changed amount"
exit 0
