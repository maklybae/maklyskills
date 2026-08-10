#!/bin/sh
#witness:red refund-bugs
# Witness for the names in a stored document: rename the price tag and the suite
# has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" x4 services/orders/internal/core/order.go \
	"{ if (\$0 ~ /UnitPrice int64/) { sub(/json:\"[a-z_]+\"/, \"json:\\\"price_minor\\\"\") } print }"
