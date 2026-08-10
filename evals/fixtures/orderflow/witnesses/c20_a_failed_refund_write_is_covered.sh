#!/bin/sh
#witness:red refund-bugs,refund-clean
# Witness for the write that records a refund: swallow its error and the suite
# has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" c20 services/orders/internal/core/refund.go \
	"{ if (\$0 ~ /return Refund\{\}, xerrors\.Wrapf\(err, \"record refund %s\", refund\.ID\)/) next; print }"
