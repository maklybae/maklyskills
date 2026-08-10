#!/bin/sh
#witness:red refund-bugs,refund-clean
# Witness for what is left of an order: give nothing back to a refund that
# names no lines once something has been refunded, and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" c25 services/orders/internal/core/refund.go \
	"{ if (\$0 ~ /^\t\treturn linesValue\(order\.Items\) - order\.Refunded\(\), nil\$/) { print \"\t\tif order.Refunded() > 0 {\"; print \"\t\t\treturn 0, nil\"; print \"\t\t}\"; print \"\t\treturn linesValue(order.Items), nil\"; next } print }"
