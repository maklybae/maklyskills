#!/bin/sh
#witness:red refund-bugs
# Witness for the return call: stop passing the request id on and the suite has
# to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" x5 services/orders/internal/inventory/client.go \
	"{ if (\$0 ~ /req\.Header\.Set\(httputil\.HeaderRequestID, id\)/) next; print }"
