---
name: flow-review
description: >-
  Run an independent multi-lens review of a change — correctness, integration
  and blast radius, production robustness, tests, security — and record
  verified, ranked findings in the task ledger. Use
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

The diff bounds what gets **read**, not what may be reported. The unit of scope
is the **touched function**: a defect in unchanged lines of a function this
change edits is in scope, because the change re-exposed it and had the chance to
fix it. Code the change never came near is not. What the wider scope must not do
is blur who introduced what, which is why every finding carries an `origin` —
the reviewer prompt owns how that is established, and it needs the profile's
`file at base` command to do it.

## 2. Build the context packet

Everything the reviewers share, assembled once:

- The **diff command** and the file list — not the diff pasted inline. They need
  to read the surrounding code anyway, so give them the means, not a snapshot.
- The **file-at-base command** from the profile's `## VCS` section, which is how
  a reviewer establishes origin instead of guessing it.
- The **task** — the ledger's `## Task`, or, before a ledger exists on the
  contained route, the route line and the three-line plan that opened the
  work — so a finding can be judged against intent instead of guessed at.
- The **profile's** stack, commands and constraints. Not its criticality:
  severity describes what a defect costs, and a reviewer told the code is
  peripheral starts discounting findings it should simply be reporting. That
  section belongs to `flow-resolve`, which decides what to do about them.
- **Previously settled findings** — every ledger finding with a verdict of
  `accepted`, `rejected` or `deferred`, verbatim with its reason. This is what
  makes the loop converge: without it a fresh reviewer re-derives the same
  suggestion every round, forever, and inherited defects are the ones it
  re-derives most reliably.
- **Out of scope**, stated plainly: comments, formatting and naming style
  (`flow-cleanup` owns them; it ran before the first review, and round fixes
  are not cleaned between rounds, so from the second round on the diff carries
  unclean fixes — a reviewer reports the defect under a comment, never the
  comment), and conformance to the written spec unless the user asked for that
  lens.

## 3. Dispatch the lenses in parallel

Pick the lens set from [references/lenses.md](references/lenses.md) — the five
defaults for a feature; two or three, named there, for a contained-route
change; more when the change, the profile's criticality, or the user calls for
it. Fill
[references/reviewer-prompt.md](references/reviewer-prompt.md) once per lens and
send **all of the Agent calls in a single message** so they run concurrently.
Dispatch each lens as a `flow-reviewer` agent — the bundle ships it, and its
tool set carries no edit tools, so read-only is a property rather than a
promise. Where the bundle's agents are not installed, fall back to
`general-purpose`; either way reviewers read and run commands, they never edit.

The prompt template carries the parts that decide review quality: findings must
be proven before they are reported, findings are capped and ranked, and finding
nothing is an acceptable answer. Do not paraphrase it — pressure to produce
findings is what manufactures false positives, and a reviewer that has to fill a
quota will fill it with noise.

## 4. Merge

- **Deduplicate.** The same defect will arrive from two lenses in two
  vocabularies. Keep the version with the concrete failure scenario.
- **Drop what is already settled.** If a finding restates one the ledger marks
  `accepted`, `rejected` or `deferred`, drop it — unless it brings genuinely new
  information, in which case say what changed.
- **Never drop a finding for being inherited.** `pre-existing` is a field, not a
  filter: the finding is recorded, and `flow-resolve` decides what it is worth.
  A defect thrown away here leaves no trace that anyone ever saw it.
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
in `../flow/references/ledger.md` relative to this skill's directory. On the
contained route this is the stage that creates it: header with the route,
`## Task` from the route line and the three-line plan, `## Verification` from
the runs already made in this session, then the findings. From here on the next
round needs it. Ids are never reused, so that "R7 is back" stays a meaningful
sentence.

Report to the user compactly — id, severity, origin, `file:line`, and the
one-line failure scenario. No praise section, no summary of what the code does:
they wrote it. If a lens found nothing, say so in one line; that is information
too.

Then hand over to `flow-resolve`, which gives every finding a verdict.

## What this stage does not do

It does not fix anything. Mixing review and repair loses the record of what was
found, and a reviewer that starts editing stops reviewing. It also does not
re-run the tests — `flow-test` did that, and its results are in the ledger.
