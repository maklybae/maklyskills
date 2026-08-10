# Shared by the witnesses that ask whether the suite would notice a change.
# Usage: mutation_witness <checkout> <label> <file> <awk program>
mutation_witness() {
	checkout=$1
	label=$2
	target=$3
	program=$4

	[ -f "$checkout/$target" ] || {
		echo "$label: $target is not in $checkout" >&2
		exit 2
	}

	work=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-witness.XXXXXX")
	trap "rm -rf \"$work\"" EXIT INT TERM
	tar -cf - -C "$checkout" --exclude .git . | tar -xf - -C "$work"

	awk "$program" "$checkout/$target" >"$work/$target"
	if cmp -s "$checkout/$target" "$work/$target"; then
		echo "$label: nothing changed, the witness cannot judge" >&2
		exit 2
	fi

	if (cd "$work" && make test >"$work/test.log" 2>&1); then
		echo "$label: the suite is green with the change in place"
		exit 1
	fi
	echo "$label: the suite caught the change"
	exit 0
}
