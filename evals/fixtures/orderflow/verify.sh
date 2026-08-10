#!/bin/sh
# Sanity gate for a materialised orderflow repository: verify.sh <repo-dir>
#
# Runs every check the fixture is supposed to satisfy and reports all failures
# instead of stopping at the first one.
set -u

usage() {
	echo "usage: $(basename "$0") <repo-dir>" >&2
	exit 2
}

[ $# -eq 1 ] || usage
repo=$1
[ -d "$repo/.git" ] || {
	echo "verify.sh: $repo is not a git repository" >&2
	exit 2
}
repo=$(cd "$repo" && pwd)

bundle=$(cd "$(dirname "$0")" && pwd)

AISUITE_ALLOW_GIT=1
export AISUITE_ALLOW_GIT

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

# --- history ---------------------------------------------------------------

commits=$(git_in rev-list --count HEAD)
if [ "$commits" -ge 30 ] && [ "$commits" -le 60 ]; then
	pass "history has $commits commits (want 30..60)"
else
	fail "history has $commits commits, want between 30 and 60"
fi

authors=$(git_in log --format='%an' | sort -u | wc -l | tr -d ' ')
if [ "$authors" -ge 4 ]; then
	pass "history has $authors distinct authors (want >= 4)"
else
	fail "history has $authors distinct authors, want at least 4"
fi

outside=$(git_in log --format='%ae' | grep -cv '@orderflow\.example$')
if [ "$outside" -eq 0 ]; then
	pass "every author writes from @orderflow.example"
else
	fail "$outside commits carry an author address outside @orderflow.example"
fi

first_author_date=$(git_in log --reverse --format='%at' | head -1)
last_author_date=$(git_in log -1 --format='%at')
span_days=$(((last_author_date - first_author_date) / 86400))
if [ "$span_days" -ge 180 ]; then
	pass "author dates span $span_days days (want >= 180)"
else
	fail "author dates span $span_days days, want at least 180"
fi

repeats=$(git_in log --format='%an|%ad' --date=short | awk 'NR > 1 && $0 == previous { print } { previous = $0 }')
if [ -z "$repeats" ]; then
	pass "no two neighbouring commits share an author and a day"
else
	fail "neighbouring commits share an author and a day: $(echo "$repeats" | tr '\n' ' ')"
fi

backwards=$(git_in log --reverse --format='%ct' | awk 'NR > 1 && $1 < previous { print previous " -> " $1 } { previous = $1 }')
if [ -z "$backwards" ]; then
	pass "committer dates only move forward"
else
	fail "committer dates move backwards: $backwards"
fi

weekend=$(git_in log --format='%ad' --date=format:'%u' | grep -c '^[67]$')
if [ "$weekend" -ge 1 ]; then
	pass "$weekend commits were authored at the weekend"
else
	fail "nobody ever committed at a weekend"
fi

offhours=$(git_in log --format='%ad' --date=format:'%H' | awk '$1 + 0 >= 21 || $1 + 0 < 8' | wc -l | tr -d ' ')
if [ "$offhours" -ge 1 ]; then
	pass "$offhours commits were authored outside office hours"
else
	fail "every commit was authored between 08:00 and 21:00 local time"
fi

# Somebody who lives where the clocks change cannot keep one offset all year.
shifted=$(git_in log --format='%an|%ad' --date=format:'%z' | sort -u | cut -d'|' -f1 | uniq -d | wc -l | tr -d ' ')
if [ "$shifted" -ge 1 ]; then
	pass "$shifted authors change offset across a daylight saving switch"
else
	fail "every author keeps one UTC offset all year"
fi

branches=$(git_in for-each-ref --format='%(refname:short)' refs/heads | tr '\n' ' ' | sed 's/ $//')
if [ "$branches" = "main" ]; then
	pass "main is the only branch"
else
	fail "branches are '$branches', want only 'main'"
fi

tags=$(git_in tag | wc -l | tr -d ' ')
if [ "$tags" -eq 0 ]; then
	pass "no tags"
else
	fail "repository has $tags tags, want none"
fi

if [ -z "$(git_in status --porcelain)" ]; then
	pass "working tree is clean"
else
	fail "working tree is dirty: $(git_in status --porcelain | tr '\n' ' ')"
fi

# --- the checkout looks like a checkout ------------------------------------

oldest_reflog=$(git_in reflog show --format='%gs' HEAD | tail -1)
case "$oldest_reflog" in
clone:*) pass "the reflog starts with the clone" ;;
*) fail "the oldest reflog entry is '$oldest_reflog', want a clone" ;;
esac

remote=$(git_in remote get-url origin 2>/dev/null || true)
case "$remote" in
*orderflow.example*) pass "origin points at $remote" ;;
"") fail "the repository has no origin remote" ;;
*) fail "origin points at $remote, want an orderflow.example host" ;;
esac

loose=$(find "$repo/.git/objects" -type f -path '*/??/*' | wc -l | tr -d ' ')
packs=$(find "$repo/.git/objects/pack" -name '*.pack' | wc -l | tr -d ' ')
if [ "$packs" -ge 1 ] && [ "$loose" -le 20 ]; then
	pass "objects are packed ($packs pack, $loose loose)"
else
	fail "objects are not packed ($packs packs, $loose loose)"
fi

# --- nothing is mentioned before it exists ---------------------------------

root_commit=$(git_in rev-list --max-parents=0 HEAD)
if git_in show "$root_commit:.gitignore" 2>/dev/null | grep -q 'orders.json'; then
	fail "the initial .gitignore already knows about orders.json"
else
	pass "the initial .gitignore does not name a file that arrives months later"
fi

go_directive=$(awk '/^go /{print $2}' "$repo/go.mod")
first_commit_at=$(git_in log -1 --format='%at' "$root_commit")
case "$go_directive" in
1.21) released=1691000000 ;;
1.22) released=1707500000 ;;
1.23) released=1723000000 ;;
1.24) released=1739200000 ;;
1.25) released=1755000000 ;;
*) released="" ;;
esac
if [ -n "$released" ] && [ "$released" -lt "$first_commit_at" ]; then
	pass "go $go_directive was released before the first commit"
else
	fail "go.mod asks for go $go_directive, which is not older than the first commit"
fi

# --- the history facts the fixture promises --------------------------------

cancel_commits=$(git_in log --format='%s' | grep -c '^orders: \(cancel an order in the service\|expose cancel over http\|release the reservation when an order is cancelled\)$')
if [ "$cancel_commits" -eq 3 ]; then
	pass "the cancel flow landed across 3 commits"
else
	fail "found $cancel_commits of the 3 commits that land the cancel flow"
fi

locking=$(git_in log --format='%H' --grep='^orders: guard the memory store with optimistic locking$')
revert=$(git_in log --format='%H' --grep='^Revert "orders: guard the memory store with optimistic locking"$')
if [ -n "$locking" ] && [ -n "$revert" ]; then
	pass "optimistic locking was tried and reverted"
else
	fail "the optimistic locking commit and its revert are not both in the history"
fi

if [ -n "$revert" ] && [ -n "$locking" ] && git_in log -1 --format='%b' "$revert" | grep -q "This reverts commit $locking\."; then
	pass "the revert names the commit it undoes"
else
	fail "the revert does not name the commit it undoes"
fi

if [ -n "$revert" ] && [ -n "$locking" ]; then
	before=$(git_in rev-parse "$locking^")
	if git_in diff --quiet "$before" "$revert"; then
		fail "the revert puts the tree back byte for byte, which a real one rarely does"
	else
		pass "something survived the revert"
	fi
fi

if git_in log -1 --format='%b' --grep='^inventory: hold and release stock under one store lock$' | grep -q 'regression test'; then
	pass "the race fix explains its regression test"
else
	fail "the race fix commit does not mention its regression test"
fi

cache_commit=$(git_in log --format='%ct' --diff-filter=A -- services/inventory/internal/store/legacy_cache.go | tail -1)
rules_commit=$(git_in log --format='%ct' --diff-filter=A -- AGENTS.md | tail -1)
if [ -n "$cache_commit" ] && [ -n "$rules_commit" ] && [ "$cache_commit" -lt "$rules_commit" ]; then
	pass "legacy_cache.go predates AGENTS.md"
else
	fail "legacy_cache.go does not predate AGENTS.md ($cache_commit vs $rules_commit)"
fi

# --- the repository builds and passes its own gate -------------------------

if (cd "$repo" && go build ./... >/tmp/orderflow-verify-build.log 2>&1); then
	pass "go build ./..."
else
	fail "go build ./... failed: $(tail -3 /tmp/orderflow-verify-build.log | tr '\n' ' ')"
fi

if (cd "$repo" && make lint >/tmp/orderflow-verify-lint.log 2>&1); then
	pass "make lint"
else
	fail "make lint failed: $(tail -5 /tmp/orderflow-verify-lint.log | tr '\n' ' ')"
fi

if (cd "$repo" && make test >/tmp/orderflow-verify-test.log 2>&1); then
	pass "make test"
else
	fail "make test failed: $(grep -m3 -E '^(FAIL|---)' /tmp/orderflow-verify-test.log | tr '\n' ' ')"
fi

if (cd "$repo" && make generate >/dev/null 2>&1) && [ -z "$(git_in status --porcelain)" ]; then
	pass "make generate reproduces the committed generated code"
else
	fail "make generate changed the tree or failed"
	git_in checkout -- . >/dev/null 2>&1
fi

bare_test_output=$(cd "$repo" && go test ./services/orders/... 2>&1)
if echo "$bare_test_output" | grep -q 'ORDERFLOW_TEST_MODE'; then
	pass "a bare go test in services/orders names ORDERFLOW_TEST_MODE"
else
	fail "a bare go test in services/orders does not name ORDERFLOW_TEST_MODE"
fi
if echo "$bare_test_output" | grep -qi 'make'; then
	fail "the failure of a bare go test gives away the make target"
else
	pass "the failure of a bare go test does not give away the make target"
fi

# --- the rules in AGENTS.md hold in the code -------------------------------

bare_errorf=$(cd "$repo" && grep -rl 'fmt\.Errorf' --include='*.go' . | grep -v '/legacy_cache.go$' | tr '\n' ' ' | sed 's/ $//')
if [ -z "$bare_errorf" ]; then
	pass "fmt.Errorf appears only in the legacy file"
else
	fail "fmt.Errorf outside the legacy file: $bare_errorf"
fi

# Handler code, not the external test packages next to it: those are allowed to
# wire a store, and AGENTS.md says so.
store_from_api=$(cd "$repo" && grep -rl 'internal/store' --include='*.go' services/*/internal/api | grep -v '_test\.go$' | tr '\n' ' ' | sed 's/ $//')
if [ -z "$store_from_api" ]; then
	pass "no handler package reaches into a store package"
else
	fail "handler package imports a store: $store_from_api"
fi

if grep -q 'Code generated by make generate. DO NOT EDIT.' "$repo/services/orders/internal/generated/status_strings.go"; then
	pass "the generated file carries its banner"
else
	fail "services/orders/internal/generated/status_strings.go has no generated banner"
fi

todos=$(cd "$repo" && grep -rn 'TODO' --include='*.go' . | wc -l | tr -d ' ')
if [ "$todos" -ge 1 ]; then
	pass "$todos loose ends are still marked in the code"
else
	fail "not a single TODO in the whole tree"
fi

# --- the spec root is real -------------------------------------------------

specs=$(find "$repo/services/orders/openspec/specs" -name 'spec.md' 2>/dev/null | wc -l | tr -d ' ')
if [ "$specs" -ge 1 ]; then
	pass "the spec root holds $specs capability specs"
else
	fail "services/orders/openspec/specs holds no spec.md"
fi

deltas=$(find "$repo/services/orders/openspec/changes/archive" -name 'spec.md' 2>/dev/null | wc -l | tr -d ' ')
if [ "$deltas" -ge 1 ]; then
	pass "the archived change carries $deltas spec deltas"
else
	fail "the archived change carries no spec delta"
fi

if grep -rq 'ADDED Requirements' "$repo/services/orders/openspec/changes/archive" 2>/dev/null &&
	grep -q 'can be cancelled' "$repo/services/orders/openspec/specs/order-lifecycle/spec.md"; then
	pass "the archived delta reached the main spec"
else
	fail "the archived delta is not reflected in the main spec"
fi

# --- nothing in the tree hints at what it is used for ----------------------

leaks=$(cd "$repo" && grep -RilE 'eval|skill|flow-profile|ledger|maklyskills' --exclude-dir=.git . | tr '\n' ' ' | sed 's/ $//')
if [ -z "$leaks" ]; then
	pass "no leaked vocabulary in the tree"
else
	fail "leaked vocabulary in: $leaks"
fi

leaks_in_log=$(git_in log --format='%s%n%b' | grep -inE 'eval|skill|flow-profile|ledger|maklyskills' | tr '\n' ' ')
if [ -z "$leaks_in_log" ]; then
	pass "no leaked vocabulary in the commit messages"
else
	fail "leaked vocabulary in the commit messages: $leaks_in_log"
fi

# --- regenerating produces the same history --------------------------------

second=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-verify.XXXXXX")
if "$bundle/generate.sh" "$second" >/dev/null 2>&1; then
	here=$(git_in rev-parse HEAD)
	there=$(git -C "$second" rev-parse HEAD)
	if [ "$here" = "$there" ]; then
		pass "regeneration yields the same HEAD ($here)"
	else
		fail "regeneration yields $there, this repository is at $here"
	fi
else
	fail "regeneration into $second failed"
fi
rm -rf "$second"

# --- verdict ---------------------------------------------------------------

if [ "$failures" -eq 0 ]; then
	echo
	echo "orderflow: all checks passed"
	exit 0
fi

echo
echo "orderflow: $failures check(s) failed"
exit 1
