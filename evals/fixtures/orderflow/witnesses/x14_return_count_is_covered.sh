#!/bin/sh
#witness:red refund-bugs
# Witness for how often the return is sent: make a failing return record itself
# twice and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

mutation_witness "$(cd "$1" && pwd)" x14 services/orders/internal/testsupport/stubs.go \
	"{ print; if (\$0 ~ /^\tif i\.restockErr != nil \{\$/) print \"\t\ti.returned = append(i.returned, ret)\" }"
