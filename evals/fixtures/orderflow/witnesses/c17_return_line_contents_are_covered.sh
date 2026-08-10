#!/bin/sh
#witness:red refund-bugs,refund-clean
# Witness for the return the orders service sends: drop the quantity from a
# return line and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" c17 services/orders/internal/inventory/client.go \
	"{ if (\$0 ~ /lines = append\(lines, linePayload\{SKU: line\.SKU, Quantity: line\.Quantity\}\)/) print \"\t\tlines = append(lines, linePayload{SKU: line.SKU})\"; else print }"
