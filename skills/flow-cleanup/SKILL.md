---
name: flow-cleanup
description: >-
  Strip the freshly written change down to what earns its place — delete
  comments and slop through the anti-slop-code pass, delete tests that would not
  fail if the behaviour they name broke, remove code the change orphaned and
  abstractions it invented for a single caller — all without changing
  behaviour. Use this skill when the user says "почисти", "убери комментарии",
  "выкинь лишние тесты", "clean this up", "refactor before review", when a
  feature has just been implemented and tested and is about to be reviewed, or
  whenever a diff has accumulated scaffolding. Prefer it over an ad-hoc tidy so
  the comment doctrine and the test-value rule are applied the same way every
  time.
---

# Flow cleanup

Two passes over the same diff, both bound by one rule: **behaviour does not
change.** Everything here is deletion, inlining, or renaming. A hunk that
changes what the program does belongs to implement or resolve — smuggling it in
under "cleanup" is how a tidy-up ships a bug.

## Run it at arm's length

Dispatch this stage to one fresh subagent instead of doing it in your own
context. The reason is the one flow-review gives for its reviewers: you wrote
this code, and the comments and scaffolding you left are the ones you still
believe in — an author deletes less than the doctrine asks, and by the end of a
long task the context doing the deleting is also the most expensive one to
spend. A fresh reader owes the diff nothing.

Dispatch it as a `flow-cleaner` agent — the bundle ships it, and its tool set
cannot dispatch agents of its own, so the recursion this section would
otherwise invite is closed by construction. Where the bundle's agents are not
installed, fall back to `general-purpose`. Give the subagent: this skill and
anti-slop-code, the profile, the diff command from its `## VCS` section, and
the ledger path for the flags. It edits, re-runs the full verification, and
returns the report from Finish below; you carry its behavioural flags into the
ledger. If you are that subagent — the dispatch named you the cleaner — skip
this section and do the passes.

## Pass 1 — the code

Run the `anti-slop-code` skill on the diff — it ships in this bundle, so when
installed as a plugin it is listed under the bundle's prefix. It owns the
comment doctrine (zero by default, a one-line cap, no doc-header on a symbol
merely for being exported) and the split between what you clean silently and
what you surface. Do not restate its rules here or improvise a lighter version —
invoke it.

Whatever it flags as *behavioural* comes back as a finding for the ledger
(`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`,
schema in `../flow/references/ledger.md` relative to this skill's directory),
not as an edit. That boundary is the whole
reason the split exists.

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

## Also in scope

- **Orphans.** Functions, exports, and branches the change left unreachable.
  Check every caller first, including tests and generated code.
- **Single-caller abstractions invented during the work** — an interface with
  one implementer and no fake behind it, a wrapper that only forwards, a
  parameter that is always the same value. Inline them.
- **Scaffolding.** Debug prints, temporary flags, commented-out code, TODOs
  nobody will do.

## Finish

Re-run the full verification. Cleanup is the stage most likely to break
something quietly, precisely because deletion feels safe.

Then report as one list: what was removed and why, with anything you flagged
rather than touched kept visibly separate.
