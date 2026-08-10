#!/bin/sh
#witness:red refund-bugs
# Witness for the order view after a refund: drop what has gone back from the
# payload and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" x13 services/orders/internal/api/orders.go \
	"{ if (\$0 ~ /^\t\tRefunded: *order\.Refunded\(\),\$/) print \"\t\tRefunded:     0,\"; else print }"
