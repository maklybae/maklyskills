#!/bin/sh
#witness:red refund-clean
# Witness for the clause that keeps an order worth nothing shippable: take it
# away and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
checkout=$(cd "$1" && pwd)

# A build with no refund clause on the ship transition has nothing to weaken.
grep -q 'value > 0 && order.Refunded() >= value' \
	"$checkout/services/orders/internal/core/service.go" || exit 0

. "$(dirname "$0")/_mutate.sh"

mutation_witness "$checkout" c23 services/orders/internal/core/service.go \
	"{ sub(/value > 0 && order\.Refunded\(\) >= value/, \"order.Refunded() >= value\"); print }"
