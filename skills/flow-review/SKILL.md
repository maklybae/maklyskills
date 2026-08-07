---
name: flow-review
description: >-
  Review a change through several independent lenses at once — correctness,
  integration and blast radius, production robustness, tests, security — by
  dispatching one read-only subagent per lens so their contexts never mix, then
  deduplicating, ranking and recording verified findings in the task ledger. Use
  this skill when the user says "проревьюй", "посмотри что не так", "ищи баги",
  "review this", "check my changes before the PR", when an implementation has
  been tested and cleaned and is about to be committed or sent for review, or
  after fixing a previous round of findings. Prefer it over reading the diff
  yourself: you wrote the code, and the blind spots that produced the bug are
  the same ones that will hide it from you.
---

# Flow review

The value of this stage comes from **independence**, not from thoroughness. You
already know what the code was meant to do, so reading it again mostly confirms
your intent. A subagent that sees only the diff, the task statement and one lens
has no such loyalty — and five of them, unable to see each other's reasoning,
will not converge on the same comfortable story.

## 1. Scope the diff

Take the base ref and the diff command from the profile's `## VCS` section and
the ledger's `base:` line. Run it. Confirm it is non-empty and that its size is
what you expect before spending five subagents on it — a bad ref discovered
inside the fan-out wastes the whole round.

## 2. Build the context packet

Everything the reviewers share, assembled once:

- The **diff command** and the file list — not the diff pasted inline. They need
  to read the surrounding code anyway, so give them the means, not a snapshot.
- The **task** — the ledger's `## Task`, so a finding can be judged against
  intent instead of guessed at.
- The **profile's** stack, commands and constraints.
- **Previously settled findings** — every ledger finding with a verdict of
  `accepted` or `rejected`, verbatim with its reason. This is what makes the
  loop converge: without it a fresh reviewer re-derives the same suggestion
  every round, forever.
- **Out of scope**, stated plainly: comments, formatting and naming style
  (`flow-cleanup` owns them), and conformance to the written spec unless the
  user asked for that lens.

## 3. Dispatch the lenses in parallel

Pick the lens set from [references/lenses.md](references/lenses.md) — the five
defaults unless the change or the user calls for more. Fill
[references/reviewer-prompt.md](references/reviewer-prompt.md) once per lens and
send **all of the Agent calls in a single message** so they run concurrently.
Use the `general-purpose` subagent type; reviewers read and run commands, they
never edit.

The prompt template carries the parts that decide review quality: findings must
be proven before they are reported, findings are capped and ranked, and finding
nothing is an acceptable answer. Do not paraphrase it — pressure to produce
findings is what manufactures false positives, and a reviewer that has to fill a
quota will fill it with noise.

## 4. Merge

- **Deduplicate.** The same defect will arrive from two lenses in two
  vocabularies. Keep the version with the concrete failure scenario.
- **Drop what is already settled.** If a finding restates one the ledger marks
  `accepted` or `rejected`, drop it — unless it brings genuinely new
  information, in which case say what changed.
- **Rank** by severity: blocker, serious, minor. The ladder is defined in the
  prompt template; use it unchanged so severities mean the same thing across
  rounds.
- **Sanity-check the blockers yourself.** A claimed blocker justifies thirty
  seconds of your own reading. If the trigger path does not hold up, demote it
  and say why — passing an unverified blocker to the user costs their trust in
  the whole stage.

## 5. Record and report

Append each surviving finding to the ledger's `## Findings` with a fresh id —
the ledger is at
`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`, schema
in `../flow/references/ledger.md` relative to this skill's directory. Ids are never reused, so that "R7 is back" stays a meaningful
sentence.

Report to the user compactly — id, severity, `file:line`, and the one-line
failure scenario. No praise section, no summary of what the code does: they
wrote it. If a lens found nothing, say so in one line; that is information too.

Then hand over to `flow-resolve`, which gives every finding a verdict.

## What this stage does not do

It does not fix anything. Mixing review and repair loses the record of what was
found, and a reviewer that starts editing stops reviewing. It also does not
re-run the tests — `flow-test` did that, and its results are in the ledger.
