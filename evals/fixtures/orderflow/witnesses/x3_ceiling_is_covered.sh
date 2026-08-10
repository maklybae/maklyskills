#!/bin/sh
#witness:red refund-bugs
# Witness for the refund ceiling: stop refusing anything and the suite has to
# notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" x3 services/orders/internal/core/refund.go \
	"{ if (\$0 ~ /^\tif amount > left \{\$/) print \"\tif false {\"; else print }"
