#!/bin/sh
#witness:red refund-bugs,refund-clean
# Witness for an order that is refunded twice: keep only the newest refund and
# the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" c19 services/orders/internal/core/refund.go \
	"{ if (\$0 ~ /^\torder\.Refunds = append\(order\.Refunds, refund\)\$/) print \"\torder.Refunds = []Refund{refund}\"; else print }"
