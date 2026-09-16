---
name: flow-test
description: >-
  Run the repository's real verification — scoped while iterating, full before
  declaring anything done — read the output, diagnose failures at the root
  rather than at the assertion, and record the evidence in the task ledger. Use
  this skill when the user says "прогони тесты", "проверь что всё зелёное",
  "run the tests", "does it build", when code has just been written or changed
  and its state is unknown, and always before claiming that work is complete,
  fixed, or passing. Prefer it over asserting from memory that something passes:
  a claim without fresh output in this session is a guess.
---

# Flow test

## Two rhythms

**Narrow while iterating.** Run the package or module you just touched, as often
as you like. Fast feedback is what keeps a mistake one edit old instead of ten.

**Broad before declaring done.** Once, at the end, run what the profile's
`## Commands` calls the full verification — plus lint and build if they are
separate. Scoped runs miss exactly the failures that matter most: the ones your
change caused somewhere you were not looking. Once means once: when the
ledger's last full line is newer than the last edit, it stands, whoever ran it,
and running it again is not diligence.

Where the language has a race detector or equivalent sanitiser, use it — **in
the scoped runs too, not only the final one**. A race caught at the stage that
introduced it is a one-edit fix; caught in the closing sweep, it surfaces after
other code has been built on top of it. Races are invisible without the
detector, and in some toolchains a single race takes down the whole test binary
and reports as unrelated neighbouring failures.

## Evidence

No claim without output produced in *this* session. "Tests should pass now" is
not a status; it is a hypothesis with the experiment skipped. Say what you ran
and what came back — counts, not adjectives.

This is not ceremony. The failure mode it prevents is specific and common: code
is changed, the change looks obviously right, the claim is made, and the suite
was never run because the last run was four edits ago.

Output produced by a subagent this task dispatched — the cleaner at its Finish,
a reviewer running a probe — and recorded in the ledger's `## Verification`
with its command and result is this session's output. Read the line; do not run
the command again to see it with your own eyes. Whether a run stands is decided
by whether it is newer than the last edit, not by who ran it, and repeating a
run that stands proves nothing the line did not.

## Diagnosing a failure

**Default assumption: the code is wrong, not the test.** A test is a recorded
expectation, usually written when the behaviour was clearer than it is now.

Changing an assertion to match new behaviour is legitimate — the requirement may
genuinely have changed — but only when you can say *why the old expectation was
wrong*. Write that sentence in the ledger. If you cannot write it, you are
editing the test to make the failure go away, which converts a caught bug into a
shipped one.

**An intermittent failure is a finding, not noise.** Re-running until green
hides a real defect that happens to be rare; concurrency bugs almost always
present this way first.

**"Pre-existing" is a claim that needs evidence.** When something fails in code
you did not touch, check whether your change reaches it before saying so — the
same one line proves or disproves it. If it really is pre-existing, note it in
the ledger and leave it alone; fixing it silently inside this change makes the
diff harder to review.

**A test green in the last full run and red now is yours.** That is the same
claim in the other direction, and it needs the same evidence before it can be
called unrelated. It matters most under repair, where a regression explained
away as flakiness is a defect the suite had already caught once.

## Record it

Append to `## Verification` in the ledger —
`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`, schema
in `../flow/references/ledger.md` relative to this skill's directory — the date,
the exact command, and the result, and mark the full runs as full.
Later stages and review subagents read those lines to know what has actually
been proven, and a stale one is worse than none. The last full line does
double duty: it is the baseline a later fix is measured against, which a scoped
run cannot be — it is green in exactly the places nobody changed.

## Not in this stage

Adding coverage belongs to `flow-implement`, deleting worthless tests belongs to
`flow-cleanup`. This stage runs what exists and repairs what it breaks — if the
run reveals a gap, note it as a finding rather than expanding scope here.
