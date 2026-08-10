#!/bin/sh
#witness:red refund-clean
# Witness for the lock the refund gate picks. Where int is 32 bits wide the
# conversion of a 32 bit digest is signed, so the model below - int32 in place
# of int - is what that build computes. The index has to land inside the stripe
# either way.
set -eu
[ $# -eq 1 ] || {
	echo "usage: $(basename "$0") <checkout>" >&2
	exit 2
}
checkout=$(cd "$1" && pwd)
target=services/orders/internal/core/refund.go

# A gate that does not convert the digest to a signed int has nothing to model.
grep -q 'int(digest.Sum32())' "$checkout/$target" || exit 0

work=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-witness.XXXXXX")
trap 'rm -rf "$work"' EXIT INT TERM
tar -cf - -C "$checkout" --exclude .git . | tar -xf - -C "$work"

awk '{ sub(/int\(digest\.Sum32\(\)\)/, "int(int32(digest.Sum32()))"); print }' \
	"$checkout/$target" >"$work/$target"

cat >"$work/services/orders/internal/core/zz_probe_test.go" <<'PROBE'
package core

import (
	"hash/fnv"
	"strconv"
	"testing"
)

func TestProbeGateIndex(t *testing.T) {
	var gate orderGate

	for i := range 100000 {
		id := "ord_" + strconv.Itoa(i)
		digest := fnv.New32a()
		_, _ = digest.Write([]byte(id))
		if digest.Sum32() < 1<<31 {
			continue
		}
		leave := gate.enter(id)
		leave()
		return
	}
	t.Skip("no order id in the sample hashes with the high bit set")
}
PROBE

if (cd "$work" && ORDERFLOW_TEST_MODE=memory go test ./services/orders/internal/core/ \
	-run '^TestProbeGateIndex$' -count=1 >"$work/probe.log" 2>&1); then
	echo "c14: the gate stays inside its stripe where int is 32 bits wide"
	exit 0
fi
echo "c14: the gate leaves its stripe where int is 32 bits wide"
exit 1
