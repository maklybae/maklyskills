---
name: flow-cleanup
description: >-
  Strip the freshly written change down to what earns its place — delete
  comments and slop through the anti-slop-code pass, delete tests that would not
  fail if the behaviour they name broke, remove code the change orphaned and
  abstractions it invented for a single caller — all without changing
  behaviour. In the pipeline it runs once before the first review and at most
  once more, narrowed to the review fixes, at the exit of flow-resolve; never
  between review rounds. Use this skill when the user says "почисти", "убери
  комментарии", "выкинь лишние тесты", "clean this up", "refactor before
  review", when a feature has just been implemented and tested and is about to
  be reviewed, or whenever a diff has accumulated scaffolding. Prefer it over an
  ad-hoc tidy so the comment doctrine and the test-value rule are applied the
  same way every time.
---

# Flow cleanup

Two passes over the same diff, both bound by one rule: **behaviour does not
change.** Everything here is deletion, inlining, or renaming. A hunk that
changes what the program does belongs to implement or resolve — smuggling it in
under "cleanup" is how a tidy-up ships a bug.

## When it runs

By route, and never between review rounds. On the feature route, twice at
most. On the contained route, once, in place, before the review. On the
mechanical route, not as a stage at all: the doctrine applies while writing,
and reading the diff once before calling it done is the whole pass.

The full pass is stage 5: after `flow-test` has gone green and before the first
review. Reviewers then read a diff without the comments that argue for the
code, the tests that only look like coverage and the branches nothing reaches;
what this pass flags becomes findings the loop actually resolves; and whatever
it breaks quietly is caught by the review that follows.

The second pass is `flow-resolve`'s exit step, and it is narrower: only the
round fixes, only when some fix touched non-test code, and cleaned in place
rather than dispatched unless the loop rewrote a substantial part of the
change. `flow-resolve` owns that decision.

Between rounds there is no cleanup. Reviewers are told that comments,
formatting and naming are out of their scope, so unclean fixes cost the next
round nothing, while a pass per round costs a context and a full verification
every time for code the next round is about to rewrite. If you are about to run
this stage with review rounds still ahead, stop.

## Run it at arm's length

Dispatch this stage to one fresh subagent instead of doing it in your own
context. The reason is the one flow-review gives for its reviewers: you wrote
this code, and the comments and scaffolding you left are the ones you still
believe in — an author deletes less than the doctrine asks, and by the end of a
long task the context doing the deleting is also the most expensive one to
spend. A fresh reader owes the diff nothing.

Dispatch when the diff is large enough for a fresh context to pay for itself:
as a default, more than roughly 200 changed lines or more than five files, and
only on the feature route — a contained-route change is cleaned in place by
definition. Below that, or when subagents are unavailable, do the two passes
yourself,
applying anti-slop-code and anti-slop-tests literally rather than a lighter
version from memory — a dispatched cleaner costs a context and a full
verification, and on a small diff that outweighs the author's blind spot.

In Codex, dispatch this stage to one fresh subagent. On Claude Code, use the
bundled `flow-cleaner` agent; elsewhere, give a fresh agent the same
constraints. Give it this skill, anti-slop-code and anti-slop-tests, the
profile, the diff command from its `## VCS` section (or the narrowed scope, for
the exit pass), and the ledger path. It edits, runs the full verification once,
writes that line into the ledger's `## Verification` itself, and returns the
report from Finish below; you carry its behavioural flags into the ledger as
findings. If you are the dispatched cleaner, skip this section and do the
passes.

**Once the cleaner's report is back, this stage is over.** Read the report and
fold its flags into the ledger — do not re-run Pass 1, Pass 2 or Finish
yourself. Everything from here down (Pass 1 through Finish) is the cleaner's
job, or yours only when the diff was small enough to clean in place — not a
checklist for the dispatcher to repeat after the fact.

## One verification per cleanup

Whoever edited runs the profile's full verification exactly once, at Finish,
and appends the result line to the ledger's `## Verification`, marked full. The
report lists every command that was run with its result — the build, any
scoped tests, the full command — and each of those is done.

The dispatcher runs none of them again. Not the build "to be sure", not the
scoped tests "because they are cheap", not the full command "to see it with my
own eyes": the ledger line is this session's evidence, the same line
`flow-test` would have written, and repeating the run proves nothing the line
did not while doubling the most expensive step of the stage. The only command
that belongs after the report is the full one, and only when the report shows
it never ran — and even then none of the scoped ones the report lists.

## Pass 1 — the code

Run the `anti-slop-code` skill on the diff — it ships in this bundle, so when
installed as a plugin it is listed under the bundle's prefix. It owns the
comment doctrine (zero comments; directives stay, and one-line invariants only
on the paths the profile's `## Comments` section lists) and the split between
what you clean silently and what you surface. Do not restate its rules here or
improvise a lighter version — invoke it.

Whatever it flags as *behavioural* comes back as a finding for the ledger
(`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`,
schema in `../flow/references/ledger.md` relative to this skill's directory),
not as an edit. That boundary is the whole
reason the split exists. Its *Relocate* list — the reasons the deleted comments
carried that found no name or test — goes into the ledger's `## Decisions`, one
line each, so it reaches the commit message and the PR description instead of
evaporating with the comments.

## Pass 2 — the tests

Run the `anti-slop-tests` skill on the same diff — it ships in this bundle
alongside `anti-slop-code`, so when installed as a plugin it is listed under the
bundle's prefix. It owns the rule of value (a test earns its place if it would
fail when a plausible bug is introduced into the behaviour it names), the
falsification probe that settles an arguable case by measurement, the catalogue
of forms, and the delete-vs-rewrite split. Do not restate its rules here or
improvise a lighter version — invoke it.

Two of its four verdicts come back to you rather than landing as edits:

- Everything it **flags** — a coverage gap the deletion exposed, a green test
  asserting wrong behaviour, a flaky or skipped test — becomes a finding for the
  ledger, exactly as with Pass 1's behavioural flags.
- Its **rewrites** are the pass's real risk. A rewrite is a code edit wearing a
  cleanup's clothes, so it is covered by the re-run below like any other.

The one rule worth repeating here because it is the one most easily skipped
under time pressure: a badly written test that is nevertheless the only coverage
of a real branch gets **rewritten, not deleted**. Deletion is correct when the
behaviour is covered elsewhere or was never worth covering; it is not correct
when it is the last thing standing between a branch and silence.

Tests that the ledger's findings name as a verdict's regression are the easy
case: they were red before their fix and green after, so they satisfy the rule
of value by construction. Keep them, and read the ledger before the pass so you
know which ones they are.

## Also in scope

- **Orphans.** Functions, exports, and branches the change left unreachable.
  Check every caller first, including tests and generated code.
- **Single-caller abstractions invented during the work** — an interface with
  one implementer and no fake behind it, a wrapper that only forwards, a
  parameter that is always the same value. Inline them.
- **Scaffolding.** Debug prints, temporary flags, commented-out code, TODOs — a
  real one goes to the ledger's open questions or the tracker, never the code.

## Finish

*(This is the cleaner's step — see "Run it at arm's length" above. If you are
the dispatcher reading this after the cleaner reported back, you already have
its verification line; do not run this again.)*

Run the profile's full verification once — build and lint too, where they are
separate commands — and append the line to the ledger's `## Verification`,
marked full. Cleanup is the stage most likely to break something quietly,
precisely because deletion feels safe, and this run is the only reader after
it.

Then report as one list: the commands you ran with their results, what was
removed and why, and anything you flagged rather than touched kept visibly
separate.
