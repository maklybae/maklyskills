#!/bin/sh
# Materialise the orderflow repository at <target-dir>.
#
#   generate.sh <target-dir>                   trunk only, main checked out
#   generate.sh <target-dir> --branch <name>   trunk plus branches/<name>, that
#                                              branch checked out on top of main
#
# The history is built commit by commit in a scratch directory and the target is
# then cloned from it, so the result looks like a checkout of a repository that
# lives somewhere else: packed objects, a reflog that starts with the clone, an
# origin remote.
#
# Every commit takes its author, committer and both dates from step.env, so two
# runs produce the same SHAs. Nothing here reads the clock or a timezone
# database; the offsets in step.env are fixed data.
set -eu

usage() {
	echo "usage: $(basename "$0") <target-dir> [--branch <name>]" >&2
	exit 2
}

[ $# -ge 1 ] || usage
target=$1
shift

branch=""
while [ $# -gt 0 ]; do
	case $1 in
	--branch)
		[ $# -ge 2 ] || usage
		branch=$2
		shift 2
		;;
	*) usage ;;
	esac
done

bundle=$(cd "$(dirname "$0")" && pwd)
steps=$bundle/steps
origin=git@git.orderflow.example:platform/orderflow.git
feature=feature/refund

if [ ! -d "$steps" ]; then
	echo "generate.sh: no steps directory at $steps" >&2
	exit 1
fi

if [ -n "$branch" ] && [ ! -d "$bundle/branches/$branch/steps" ]; then
	echo "generate.sh: no branch called '$branch' under $bundle/branches" >&2
	exit 1
fi

if [ -e "$target" ] && [ -n "$(ls -A "$target" 2>/dev/null || true)" ]; then
	echo "generate.sh: $target exists and is not empty, refusing to overwrite it" >&2
	exit 1
fi

# Ignore whatever this machine configures for git: templates, hooks, signing and
# autocrlf all change the objects we are about to write.
GIT_CONFIG_GLOBAL=/dev/null
GIT_CONFIG_SYSTEM=/dev/null
export GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM

# Some sandboxes wrap the git command; this lets the wrapper through.
AISUITE_ALLOW_GIT=1
export AISUITE_ALLOW_GIT

scratch=$(mktemp -d "${TMPDIR:-/tmp}/orderflow-history.XXXXXX")
cleanup() {
	rm -rf "$scratch"
}
trap cleanup EXIT INT TERM

work=$scratch/orderflow
shas=$scratch/shas
mkdir -p "$work" "$shas"

commits=0

apply_steps() {
	for step in "$1"/*/; do
		[ -f "$step/step.env" ] || {
			echo "generate.sh: $step has no step.env" >&2
			exit 1
		}

		AUTHOR_NAME=""
		AUTHOR_EMAIL=""
		AUTHOR_DATE=""
		COMMIT_DATE=""
		# shellcheck disable=SC1091
		. "$step/step.env"

		if [ -d "$step/tree" ]; then
			(cd "$step/tree" && find . -type f) | while read -r file; do
				mkdir -p "$work/$(dirname "$file")"
				cp "$step/tree/$file" "$work/$file"
				chmod 644 "$work/$file"
			done
		fi

		if [ -f "$step/delete.txt" ]; then
			while read -r path; do
				[ -n "$path" ] || continue
				rm -f "$work/$path"
			done <"$step/delete.txt"
		fi

		# A revert names the commit it undoes, the way git writes it.
		message=$scratch/message.txt
		if [ -f "$step/revert-of" ]; then
			reverted=$(cat "$shas/$(cat "$step/revert-of")")
			sed "s/__REVERT_SHA__/$reverted/" "$step/message.txt" >"$message"
		else
			cp "$step/message.txt" "$message"
		fi

		git -C "$work" add -A
		GIT_AUTHOR_NAME=$AUTHOR_NAME \
			GIT_AUTHOR_EMAIL=$AUTHOR_EMAIL \
			GIT_AUTHOR_DATE=$AUTHOR_DATE \
			GIT_COMMITTER_NAME=$AUTHOR_NAME \
			GIT_COMMITTER_EMAIL=$AUTHOR_EMAIL \
			GIT_COMMITTER_DATE=$COMMIT_DATE \
			git -C "$work" -c commit.gpgsign=false commit -q -F "$message"

		slug=$(basename "${step%/}")
		git -C "$work" rev-parse HEAD >"$shas/${slug#*-}"
		commits=$((commits + 1))
	done
}

git init -q -b main "$work"
apply_steps "$steps"

if [ -n "$branch" ]; then
	git -C "$work" checkout -q -b "$feature"
	apply_steps "$bundle/branches/$branch/steps"
fi

# file:// rather than a plain path: a local path clone hardlinks loose objects,
# the transport sends a packfile.
GIT_AUTHOR_NAME="Dan Ruiz" GIT_AUTHOR_EMAIL="druiz@orderflow.example" \
	GIT_COMMITTER_NAME="Dan Ruiz" GIT_COMMITTER_EMAIL="druiz@orderflow.example" \
	git clone -q "file://$work" "$target"

target=$(cd "$target" && pwd)
git -C "$target" remote set-url origin "$origin"

# The branch is what a reviewer works on; main is the base it is measured
# against, so both have to exist locally.
if [ -n "$branch" ]; then
	git -C "$target" branch -q --track main origin/main
fi

# The scratch path is an artefact of how this was built; the reflog should read
# like a clone from the remote the checkout now points at.
for log in "$target/.git/logs/HEAD" \
	"$target/.git/logs/refs/heads/main" \
	"$target/.git/logs/refs/heads/$feature" \
	"$target/.git/logs/refs/remotes/origin/HEAD"; do
	[ -f "$log" ] || continue
	sed "s|clone: from .*|clone: from $origin|" "$log" >"$log.rewritten"
	mv "$log.rewritten" "$log"
done

echo "orderflow: $commits commits, HEAD $(git -C "$target" rev-parse HEAD) at $target"
