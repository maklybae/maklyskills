#!/bin/sh
#witness:red refund-bugs,refund-clean
# Witness for a return inventory refuses: book the lines onto the shelf before
# the request is judged and the suite has to notice.
set -eu
[ $# -eq 1 ] || { echo "usage: $(basename "$0") <checkout>" >&2; exit 2; }
. "$(dirname "$0")/_mutate.sh"

program='
/^func \(s \*Service\) Return\(ctx context\.Context, req ReturnRequest\) \(\[\]Level, error\) \{$/ {
	print
	print "\treq.OrderID = strings.TrimSpace(req.OrderID)"
	print ""
	print "\tlevels, err := s.store.ReturnLines(ctx, req.Lines)"
	print "\tif bad := req.validate(); bad != nil {"
	print "\t\treturn nil, bad"
	print "\t}"
	print "\tif err != nil {"
	print "\t\treturn nil, xerrors.Wrapf(err, \"take the return of order %s back\", req.OrderID)"
	print "\t}"
	print "\treturn levels, nil"
	inside = 1
	next
}
inside && /^\}$/ { inside = 0; print; next }
inside { next }
{ print }
'

mutation_witness "$(cd "$1" && pwd)" c18 services/inventory/internal/core/service.go "$program"
