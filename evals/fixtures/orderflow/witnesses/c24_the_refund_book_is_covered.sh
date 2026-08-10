#!/bin/sh
#witness:red refund-clean
# Witness for the refund book beside the order document: write only the first
# refund of an order into it and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
checkout=$(cd "$1" && pwd)

# A build that keeps refunds in the order document alone has no book to spoil.
grep -q 'refundsPath' "$checkout/services/orders/internal/store/jsonfile.go" || exit 0

. "$(dirname "$0")/_mutate.sh"

mutation_witness "$checkout" c24 services/orders/internal/store/jsonfile.go \
	"{ sub(/refunds\[id\] = order\.Refunds\$/, \"refunds[id] = order.Refunds[:1]\"); print }"
