# The task ledger

One file per task, **outside the repository**:

```
~/.claude/projects/<project-root-with-slashes-as-dashes>/flow/<task-slug>/ledger.md
```

`<project-root-with-slashes-as-dashes>` is the absolute path of the project root
with every `/` replaced by `-` — the same convention Claude Code already uses
for its own per-project directories, so the ledger lands beside them. The
project root is the directory that holds `.claude/flow-profile.md`; anchoring to
it rather than to the current directory keeps the path stable whether the
session started at the top of the tree or three levels down inside a service.

`<task-slug>` is a kebab-case name for the task: `add-vector-index`,
`fix-retry-storm`.

**Why not in the repository.** The ledger is personal working state — dead ends,
rejected findings, "we consciously decided not to fix this". In a shared tree
that is both noise in everyone's `status` output and something you would
eventually self-censor to keep it presentable, which would destroy its value.
The profile is the opposite — objective, about the repository, useful to anyone
— so that one does live in the repository.

It is also deliberately not inside a spec directory. A spec artifact — an
openspec change, an ADR, a PRD — ships with the pull request and is read by
other people. Keep them apart and link the ledger to the spec.

## What it is for

Three consumers, none of whom share your context:

- **A later session**, after compaction or a night's sleep, that must resume
  without re-reading the whole diff.
- **A review subagent**, which needs to know what was already consciously
  accepted so it does not raise it again — without that, the review/resolve loop
  never converges.
- **The user**, who wants to see decisions and open findings without scrolling
  a transcript.

Anything that does not serve one of those three does not belong in the file.
It is a ledger, not a diary: no narration of what you did, no restating the diff.

## Schema

```markdown
# Add vector index to infra events

- slug: add-vector-index
- stage: review (round 2)
- mode: unattended, ends at commit                              # omit when interactive
- branch: users/mdk/add-vector-index
- base: trunk@a1b2c3d
- spec: openspec change `add-vector-index` in backend/harness   # or: none

## Task
Two to five sentences: what is being built and why. Written during explore,
edited only when the scope actually changes.

## Decisions
- D1 (plan) Bytes in S3, metadata in YDB. A single YDB blob column was rejected:
  rows would exceed the 8 MB limit for real documents.
- D2 (implement) Reused `sqx.TransactionManager` rather than a new helper.

## Open questions
- Q1 Does the sync worker need its own lease, or can it reuse the keeper's?
  → answered 2026-08-07: reuse the keeper's.
- Q2 ASSUMED (unattended): retention defaults to 30 days, matching the
  neighbouring table. Nobody was available to confirm; the conservative reading
  was taken and this leads the final report.

## Findings
### R4 · blocker · correctness · backend/vfs/repository.go:88
Two concurrent Upserts on the same key lose the newer row: read-modify-write
outside a transaction.
Verdict: fixed — wrapped in `TransactionManager`, test `TestUpsertConcurrent`.

### R5 · minor · tests · backend/vfs/repository_test.go:12
No case for an empty digest.
Verdict: accepted — the only caller validates the digest at the transport layer,
so a test here pins a state that cannot occur.

## Verification
- 2026-08-07 `ya test -t backend/vfs --race` → 41 passed, 0 failed
```

## Rules that matter

**Findings are append-only.** Never delete one, never rewrite its text. A
finding with `Verdict: accepted` is the record that stops the next review round
from raising it again; delete it and the loop restarts. Adding the verdict to an
existing entry is the only edit.

**Every verdict carries its reason on the same line.** `accepted` alone is
indistinguishable from `forgot to fix`. The reason is what a reviewer — human or
subagent — reads to decide whether the acceptance still holds after the code
moved.

**Finding ids are stable and never reused.** `R7` means one thing forever, so
that "R7 came back" is a meaningful sentence.

**Decisions record the road not taken.** A decision without its rejected
alternative is just a description of the code, which the code already provides.

**An assumption taken because nobody could be asked is marked `ASSUMED`.** In an
unattended run this is the highest-value line in the file: it is the one thing
the user cannot reconstruct from the diff, and it is what the closing report
leads with.

**Keep it short.** If the ledger is longer than the diff it describes, it has
started narrating. Trim.
