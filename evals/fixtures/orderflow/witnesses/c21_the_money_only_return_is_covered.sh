#!/bin/sh
#witness:red refund-bugs,refund-clean
# Witness for a refund that names no lines: send its return to inventory after
# all and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" c21 services/orders/internal/core/refund.go \
	"{ if (\$0 ~ /A refund that names no lines is money only/) { skip = 3; next } if (skip > 0) { skip--; next } print }"
