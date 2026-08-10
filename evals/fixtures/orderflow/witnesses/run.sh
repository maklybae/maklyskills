#!/bin/sh
# Run the witnesses against a materialised checkout.
#
#   run.sh <checkout> [witness ...]
#
# A witness is green (exit 0) when the property it describes holds, and red when
# the defect it was written for is present. With no witness named, all of them
# run. The checkout is left exactly as it was found.
#
# Which arms a witness is meant to be red on is declared in the witness itself,
# as a `witness:red <arm[,arm]|none>` header line; verify-branch.sh reads it and
# decides. This runner only reports what it found.
set -u

usage() {
	echo "usage: $(basename "$0") <checkout> [witness ...]" >&2
	exit 2
}

[ $# -ge 1 ] || usage
checkout=$(cd "$1" && pwd) || usage
shift

here=$(cd "$(dirname "$0")" && pwd)
ORDERFLOW_TEST_MODE=memory
export ORDERFLOW_TEST_MODE

if [ $# -gt 0 ]; then
	witnesses=$*
else
	witnesses=$(cd "$here" && ls [bcx]*_*.go [bcx]*_*.sh 2>/dev/null)
fi

red=0
green=0

for witness in $witnesses; do
	file=$here/$witness
	[ -f "$file" ] || {
		echo "?    $witness (not in $here)"
		red=$((red + 1))
		continue
	}

	case $witness in
	*.sh)
		if sh "$file" "$checkout" >/dev/null 2>&1; then
			echo "green $witness"
			green=$((green + 1))
		else
			echo "RED   $witness"
			red=$((red + 1))
		fi
		;;
	*.go)
		dest=$(awk '/^\/\/witness:dest /{print $2; exit}' "$file")
		name=$(awk '/^func Test/{sub(/\(.*/, "", $2); print $2; exit}' "$file")
		if [ -z "$dest" ] || [ -z "$name" ]; then
			echo "?    $witness (no destination or no test function)"
			red=$((red + 1))
			continue
		fi

		copied=$checkout/$dest/zz_witness_test.go
		cp "$file" "$copied"
		if (cd "$checkout" && go test "./$dest/" -run "^$name\$" -count=1 >/dev/null 2>&1); then
			echo "green $witness"
			green=$((green + 1))
		else
			echo "RED   $witness"
			red=$((red + 1))
		fi
		rm -f "$copied"
		;;
	*)
		echo "?    $witness (unknown kind)"
		red=$((red + 1))
		;;
	esac
done

echo
echo "$green green, $red red"
[ "$red" -eq 0 ]
