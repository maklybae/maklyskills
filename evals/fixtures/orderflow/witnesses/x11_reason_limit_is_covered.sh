#!/bin/sh
#witness:red refund-bugs
# Witness for the reason limit the change argued for: shorten it to the cancel
# flows and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" x11 services/orders/internal/core/refund.go \
	"{ if (\$0 ~ /^const maxRefundReasonLen = /) print \"const maxRefundReasonLen = 240\"; else print }"
