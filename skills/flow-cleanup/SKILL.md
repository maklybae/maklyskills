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

## Pass 1 — the code

Run the `anti-slop-code` skill on the diff — it ships in this bundle, so when
installed as a plugin it is listed under the bundle's prefix. It owns the
comment doctrine (zero by default, a one-line cap, no doc-header on a symbol
merely for being exported) and the split between what you clean silently and
what you surface. Do not restate its rules here or improvise a lighter version —
invoke it.

Whatever it flags as *behavioural* comes back as a finding for the ledger
(`.claude/flow/<slug>/ledger.md`, schema in `../flow/references/ledger.md`
relative to this skill's directory), not as an edit. That boundary is the whole
reason the split exists.

## Pass 2 — the tests

One rule decides every case:

> **A test earns its place if it would fail when a plausible bug is introduced
> into the behaviour it names.**

Apply it by naming the bug out loud. If no bug you can describe would turn this
test red, it is not testing anything — it is describing the implementation back
to itself, and it will need updating every time the implementation moves.

**Delete:**

- Assertions on incidental output: log lines, the wording of a message, the
  order of fields in a serialisation nobody parses.
- Restatements of the implementation: a mock is told to return `X` and the test
  asserts `X` came back; a call is verified with the arguments the test itself
  just passed in.
- Tests of the language, the framework, or a library: that a constructor assigns
  the fields you gave it, that a getter returns its field, that the standard
  library works.
- Duplicate rows in a table test that walk the same branch with different data.
  Distinct data through one path is one case, not five.
- Tests pinned to an internal detail when the same behaviour is already covered
  through the public entry point.

**Keep, and strengthen:**

- Contracts other code depends on — the module's observable behaviour.
- Error branches, especially the ones that are awkward to trigger. Those are
  exactly the paths nobody exercises by hand.
- Boundaries: empty, zero, one, maximum, expired, malformed, concurrent.
- Round trips: persistence, serialisation, backward compatibility with data that
  already exists.
- Regressions with a history. A test that exists because something once broke
  earns its place permanently; make sure its name says what it protects.

**The case that needs care** is a badly written test that is nevertheless the
only coverage of a real branch. Rewrite it — do not delete it. Deletion is
correct when the behaviour is covered elsewhere or was never worth covering; it
is not correct when it is the last thing standing between a branch and silence.

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
