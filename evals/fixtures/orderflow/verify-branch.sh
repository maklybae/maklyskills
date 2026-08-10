#!/bin/sh
# Sanity gate for a materialised feature branch:
#
#   verify-branch.sh <repo-dir> <branch-name>
#
# <branch-name> is the directory under branches/, refund-bugs or refund-clean.
# Reports every failure instead of stopping at the first one.
#
# The control arm is known, not clean: both arms carry recorded defects and the
# answer key says which. Every gate below reads that key rather than assuming
# one arm is free of everything.
set -u

usage() {
	echo "usage: $(basename "$0") <repo-dir> <refund-bugs|refund-clean>" >&2
	exit 2
}

[ $# -eq 2 ] || usage
repo=$1
branch=$2
[ -d "$repo/.git" ] || {
	echo "verify-branch.sh: $repo is not a git repository" >&2
	exit 2
}
repo=$(cd "$repo" && pwd)
bundle=$(cd "$(dirname "$0")" && pwd)

[ -d "$bundle/branches/$branch/steps" ] || usage

AISUITE_ALLOW_GIT=1
export AISUITE_ALLOW_GIT

# The trunk this material was built against. A branch that moves it is a branch
# that invalidates every earlier eval run.
TRUNK_HEAD=7dd9e5248bf3219e50af7e99da60dcda4a511bd8

key=$bundle/ground-truth/refund-bugs.json
witnesses=$bundle/witnesses

failures=0

pass() {
	echo "ok   $1"
}

fail() {
	echo "FAIL $1"
	failures=$((failures + 1))
}

git_in() {
	git -C "$repo" "$@"
}

# --- the branch sits on the trunk ------------------------------------------

head_branch=$(git_in rev-parse --abbrev-ref HEAD)
if [ "$head_branch" = "feature/refund" ]; then
	pass "feature/refund is checked out"
else
	fail "HEAD is on '$head_branch', want feature/refund"
fi

main_tip=$(git_in rev-parse main 2>/dev/null || true)
if [ "$main_tip" = "$TRUNK_HEAD" ]; then
	pass "main is the accepted trunk ($TRUNK_HEAD)"
else
	fail "main is at '$main_tip', want the accepted trunk $TRUNK_HEAD"
fi

base=$(git_in merge-base main HEAD 2>/dev/null || true)
if [ -n "$base" ] && [ "$base" = "$main_tip" ]; then
	pass "main is the merge base of the branch"
else
	fail "merge base is '$base', want main at '$main_tip'"
fi

commits=$(git_in rev-list --count main..HEAD)
if [ "$commits" -ge 3 ] && [ "$commits" -le 5 ]; then
	pass "the branch is $commits commits (want 3..5)"
else
	fail "the branch is $commits commits, want between 3 and 5"
fi

authors=$(git_in log --format='%an' main..HEAD | sort -u | wc -l | tr -d ' ')
if [ "$authors" -ge 1 ] && [ "$authors" -le 2 ]; then
	pass "the branch has $authors authors (want 1 or 2)"
else
	fail "the branch has $authors authors, want one or two"
fi

outside=$(git_in log --format='%ae' main..HEAD | grep -cv '@orderflow\.example$')
if [ "$outside" -eq 0 ]; then
	pass "every branch author writes from @orderflow.example"
else
	fail "$outside branch commits carry an address outside @orderflow.example"
fi

trunk_last=$(git_in log -1 --format='%at' main)
branch_first=$(git_in log --reverse --format='%at' main..HEAD | head -1)
if [ "$branch_first" -gt "$trunk_last" ]; then
	pass "the branch was written after the trunk it sits on"
else
	fail "the first branch commit is not later than the trunk tip"
fi

backwards=$(git_in log --reverse --format='%ct' main..HEAD | awk 'NR > 1 && $1 < previous { print previous " -> " $1 } { previous = $1 }')
if [ -z "$backwards" ]; then
	pass "branch committer dates only move forward"
else
	fail "branch committer dates move backwards: $backwards"
fi

subjects=$(git_in log --format='%s' main..HEAD | grep -cvE '^(orders|inventory|docs|ci|chore|pkg|tools): ')
if [ "$subjects" -eq 0 ]; then
	pass "branch subjects keep the repository's message style"
else
	fail "$subjects branch subjects break the message style"
fi

# --- it builds and its own suite is green ----------------------------------

if (cd "$repo" && make lint >/tmp/orderflow-branch-lint.log 2>&1); then
	pass "make lint"
else
	fail "make lint failed: $(tail -5 /tmp/orderflow-branch-lint.log | tr '\n' ' ')"
fi

if (cd "$repo" && make test >/tmp/orderflow-branch-test.log 2>&1); then
	pass "make test"
else
	fail "make test failed: $(grep -m3 -E '^(FAIL|---)' /tmp/orderflow-branch-test.log | tr '\n' ' ')"
fi

if [ -z "$(git_in status --porcelain)" ]; then
	pass "working tree is clean"
else
	fail "working tree is dirty: $(git_in status --porcelain | tr '\n' ' ')"
fi

# --- nothing in the tree says what is being measured -----------------------

answers=$(cd "$repo" && grep -RilE 'planted|decoy|witness|ground.?truth|seeded|adjudicat|answer.?key' --exclude-dir=.git . | tr '\n' ' ' | sed 's/ $//')
if [ -z "$answers" ]; then
	pass "no answer-key vocabulary in the tree"
else
	fail "answer-key vocabulary in: $answers"
fi

ids=$(cd "$repo" && grep -RlE '(^|[^A-Za-z0-9])[BCDX][0-9]+([^A-Za-z0-9]|$)' --exclude-dir=.git . | tr '\n' ' ' | sed 's/ $//')
if [ -z "$ids" ]; then
	pass "no record ids in the tree"
else
	fail "something that reads like a record id in: $ids"
fi

leaks=$(cd "$repo" && grep -RilE 'eval|skill|flow-profile|ledger|maklyskills' --exclude-dir=.git . | tr '\n' ' ' | sed 's/ $//')
if [ -z "$leaks" ]; then
	pass "no leaked vocabulary in the tree"
else
	fail "leaked vocabulary in: $leaks"
fi

leaks_in_log=$(git_in log --format='%s%n%b' main..HEAD | grep -inE 'planted|decoy|witness|ground.?truth|seeded|adjudicat|eval|skill|ledger|maklyskills' | tr '\n' ' ')
if [ -z "$leaks_in_log" ]; then
	pass "no leaked vocabulary in the branch messages"
else
	fail "leaked vocabulary in the branch messages: $leaks_in_log"
fi

stray=$(cd "$repo" && find . -path ./.git -prune -o \( -name '*witness*' -o -name 'refund-bugs.json' -o -name 'matching-notes.md' -o -name 'adjudications.md' \) -print | tr '\n' ' ')
if [ -z "$stray" ]; then
	pass "no witness or answer-key file in the checkout"
else
	fail "material that belongs in the bundle is in the checkout: $stray"
fi

# --- the answer key and the witnesses agree --------------------------------

# records prints one line per record: id, kind, witness path, arms. The key is
# generated with a fixed layout, and the count check below is what notices if
# that ever stops being true.
records() {
	awk '
		function value(line) {
			sub(/^[^:]*: "/, "", line)
			sub(/",?$/, "", line)
			return line
		}
		/^    "id": "/ { id = value($0); next }
		/^    "kind": "/ { kind = value($0); next }
		/^    "witness": "/ { witness = value($0); next }
		/^    "arms": \[/ { arms = ""; inside = 1; next }
		inside && /^    \],?$/ {
			inside = 0
			print id "\t" kind "\t" witness "\t" arms
			next
		}
		inside && /"/ {
			one = $0
			gsub(/[ \t",]/, "", one)
			arms = (arms == "" ? one : arms "," one)
		}
	' "$key"
}

# sorted turns a comma separated list into a canonical one.
sorted() {
	echo "$1" | tr ',' '\n' | sort | tr '\n' ',' | sed 's/,$//'
}

# declared answers with the arms a witness says it is red on.
declared() {
	awk '/witness:red /{ sub(/^.*witness:red[ \t]+/, ""); gsub(/[ \t\r]/, ""); print; exit }' "$1"
}

key_records=$(records)
key_count=$(echo "$key_records" | grep -c .)
id_count=$(grep -c '^    "id": "' "$key")
if [ "$key_count" -eq "$id_count" ] && [ "$id_count" -gt 0 ]; then
	pass "the answer key reads as $id_count records"
else
	fail "the answer key has $id_count records and $key_count could be read, so its layout has changed"
fi

referenced=$(echo "$key_records" | awk -F'\t' '$3 != "" { print $3 }' | sort -u)

problems=$(echo "$key_records" | awk -F'\t' -v dir="$bundle" '
	{
		id = $1; kind = $2; witness = $3; arms = $4
		if (kind == "decoy") {
			if (witness != "") print id ": a decoy carries a witness"
			next
		}
		if (witness == "") { print id ": no witness"; next }
		path = dir "/" witness
		if ((getline line < path) < 0) { print id ": " witness " is not in the bundle"; close(path); next }
		close(path)
	}
')
if [ -z "$problems" ]; then
	pass "every bug record names a witness that exists, every decoy names none"
else
	fail "answer key: $(echo "$problems" | tr '\n' ' ')"
fi

mismatched=""
for witness in $(echo "$key_records" | awk -F'\t' '$2 == "bug" { print $1 "=" $3 "=" $4 }'); do
	id=${witness%%=*}
	rest=${witness#*=}
	path=${rest%%=*}
	arms=${rest#*=}
	[ -f "$bundle/$path" ] || continue
	said=$(declared "$bundle/$path")
	if [ "$(sorted "$said")" = "$(sorted "$arms")" ]; then
		:
	else
		mismatched="$mismatched $id(declares '$said', recorded '$arms')"
	fi
done
if [ -z "$mismatched" ]; then
	pass "every witness declares the arms its record names"
else
	fail "witness and record disagree:$mismatched"
fi

orphans=""
for file in $(cd "$witnesses" && ls [bcx]*_*.go [bcx]*_*.sh 2>/dev/null); do
	said=$(declared "$witnesses/$file")
	if [ -z "$said" ]; then
		orphans="$orphans $file(no witness:red header)"
		continue
	fi
	[ "$said" = "none" ] && continue
	echo "$referenced" | grep -qx "witnesses/$file" || orphans="$orphans $file(red on $said, named by no record)"
done
if [ -z "$orphans" ]; then
	pass "every witness is either a record's or declares itself a pin"
else
	fail "witnesses without a record:$orphans"
fi

# --- the two branches are the same change ----------------------------------

# Files where the arms cannot carry the same number of lines by construction:
# one arm holds coverage or a mechanism the other must lack. Each cap is
# measured on the material, and widening one is a decision, not a repair.
shape_cap() {
	case $1 in
	services/orders/internal/core/refund_test.go) echo 38 ;;
	services/orders/internal/inventory/client_test.go) echo 30 ;;
	services/orders/internal/store/jsonfile.go) echo 26 ;;
	services/orders/internal/store/store_test.go) echo 16 ;;
	services/orders/internal/api/orders.go) echo 12 ;;
	services/orders/internal/core/refund.go) echo 12 ;;
	*) echo 10 ;;
	esac
}

sibling=refund-clean
[ "$branch" = "refund-clean" ] && sibling=refund-bugs

if [ -d "$bundle/branches/$sibling/steps" ]; then
	other=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-sibling.XXXXXX")
	rm -rf "$other"
	if "$bundle/generate.sh" "$other" --branch "$sibling" >/dev/null 2>&1; then
		here_files=$(git_in diff --name-only main..HEAD | sort)
		there_files=$(git -C "$other" diff --name-only main..HEAD | sort)
		if [ "$here_files" = "$there_files" ]; then
			pass "both branches touch the same files ($(echo "$here_files" | wc -l | tr -d ' '))"
		else
			fail "the two branches touch different files"
		fi

		here_subjects=$(git_in log --format='%s' main..HEAD)
		there_subjects=$(git -C "$other" log --format='%s' main..HEAD)
		if [ "$here_subjects" = "$there_subjects" ]; then
			pass "both branches carry the same commit subjects"
		else
			fail "the two branches tell different stories in their subjects"
		fi

		over=""
		widest=0
		for file in $here_files; do
			[ -f "$repo/$file" ] || continue
			[ -f "$other/$file" ] || continue
			mine=$(wc -l <"$repo/$file" | tr -d ' ')
			theirs=$(wc -l <"$other/$file" | tr -d ' ')
			spread=$(awk -v x="$mine" -v y="$theirs" 'BEGIN { d = x - y; if (d < 0) d = -d; m = (x > y ? x : y); if (m == 0) print 0; else print int(100 * d / m) }')
			cap=$(shape_cap "$file")
			[ "$spread" -gt "$widest" ] && widest=$spread
			if [ "$spread" -gt "$cap" ]; then
				over="$over $file($spread% over $cap%)"
			fi
		done
		if [ -z "$over" ]; then
			pass "every file is within its shape budget (widest $widest%)"
		else
			fail "files outside their shape budget:$over"
		fi
	else
		fail "could not materialise $sibling for comparison"
	fi
	rm -rf "$other"
fi

# --- the witnesses say what the branch is ----------------------------------

if [ -x "$witnesses/run.sh" ]; then
	report=$("$witnesses/run.sh" "$repo" 2>&1)
	wrong=""
	checked=0
	for file in $(cd "$witnesses" && ls [bcx]*_*.go [bcx]*_*.sh 2>/dev/null); do
		said=$(declared "$witnesses/$file")
		want=green
		case ",$(sorted "$said")," in
		*",$branch,"*) want=RED ;;
		esac

		if echo "$report" | grep -q "^RED   $file\$"; then
			got=RED
		elif echo "$report" | grep -q "^green $file\$"; then
			got=green
		else
			got=missing
		fi

		checked=$((checked + 1))
		[ "$got" = "$want" ] || wrong="$wrong $file(want $want, is $got)"
	done
	if [ -z "$wrong" ]; then
		pass "all $checked witnesses stand where the answer key puts them"
	else
		fail "witness parity on $branch:$wrong"
	fi
else
	fail "witnesses/run.sh is not executable"
fi

# --- regenerating produces the same branch ---------------------------------

second=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-branch.XXXXXX")
rm -rf "$second"
if "$bundle/generate.sh" "$second" --branch "$branch" >/dev/null 2>&1; then
	here=$(git_in rev-parse HEAD)
	there=$(git -C "$second" rev-parse HEAD)
	if [ "$here" = "$there" ]; then
		pass "regeneration yields the same tip ($here)"
	else
		fail "regeneration yields $there, this checkout is at $here"
	fi
else
	fail "regeneration of $branch failed"
fi
rm -rf "$second"

# --- verdict ---------------------------------------------------------------

if [ "$failures" -eq 0 ]; then
	echo
	echo "$branch: all checks passed"
	exit 0
fi

echo
echo "$branch: $failures check(s) failed"
exit 1
