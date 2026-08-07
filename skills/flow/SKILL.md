---
name: flow
description: >-
  The seven-stage development pipeline — explore, plan, implement, test, cleanup,
  review, resolve — plus the task ledger that carries state between stages so no
  work is lost to compaction, a new session, or a subagent boundary. Use this
  skill whenever the user wants a task taken end-to-end rather than a one-off
  edit: when they say "по флоу", "по пайплайну", "прогони весь цикл", "доведи до
  конца", "start a full cycle"; when they ask which stage comes next or want to
  resume a task started earlier; or when they hand over a ticket or feature big
  enough that plan-then-build-then-review is the honest shape of the work. Also
  use it to pick which single flow-* stage skill applies when only part of the
  cycle is wanted.
---

# Flow

## Why a pipeline and not a checklist

Long tasks fail two ways. Context evaporates — compaction, a new session, a
subagent that never saw the conversation — and the work has to be reconstructed
from the diff. And the quality steps that were supposed to happen "at the end"
never happen, because by then the work already looks done.

The pipeline answers both with one rule: **every stage ends in a written
artifact, not in accumulated context.** The artifact is the handoff. A stage
that leaves its conclusions only in the conversation has not finished.

The shared artifact is the **ledger** — one file per task, described in
[references/ledger.md](references/ledger.md). Read it when you start, update it
when you finish a stage. It is what lets a cold session, or a reviewer subagent
that knows nothing about you, act correctly.

## The stages

| # | Skill | Done when |
|---|-------|-----------|
| 1 | `flow-explore` | You can name the files you would touch and the existing code you would imitate |
| 2 | `flow-plan` | A staged plan (or an openspec change) exists and the user approved it |
| 3 | `flow-implement` | Every stage of the plan is built and its own checks pass |
| 4 | `flow-test` | The project's verification passes, with output produced in this session |
| 5 | `flow-cleanup` | Slop and worthless tests are gone, behaviour unchanged, tests still green |
| 6 | `flow-review` | Findings from parallel lenses, each verified, ranked, written to the ledger |
| 7 | `flow-resolve` | Every finding has a verdict; the fixes are applied and verified |

Stages 6 and 7 repeat. The loop stops when a full review round adds no new
blocker or serious finding — not when you run out of patience, and not after a
fixed number of rounds. `flow-resolve` owns the convergence rule.

## Entering, resuming, skipping

**New task** — start at `flow-explore`; it creates the ledger.

**Resuming** — read `.claude/flow/<slug>/ledger.md`. Its `stage:` line says
where the work stopped and its findings say what is still open. Do not
reconstruct state from the diff when a ledger exists.

**A single stage** — if the user names one ("проревьюй", "почисти"), run just
that skill. Still update the ledger; a stage run out of band is still state.

**Skipping** — a change small enough to hold in your head does not need a plan,
and a throwaway script does not need a review. But say out loud which stages you
skipped and why, so the user can disagree. Two skips are almost never right:
`flow-test`, because an unverified claim is worse than no claim, and
`flow-review` on anything that will run in production.

## Where the human is required

Three points genuinely need the user, and stopping anywhere else wastes their
time:

- **After the plan.** This is the cheapest place to change direction and the
  most expensive place to be wrong.
- **Before accepting a blocker finding.** Consciously shipping a known serious
  defect is the user's call, never yours.
- **Before anything outward-facing** — commit, push, PR. The profile's VCS
  section says *how*; the user says *whether*.

## Project-specific facts

Nothing in these skills hardcodes a build system, a VCS, or a test runner. Those
live in `.claude/flow-profile.md`, written once by `flow-setup`. If a stage needs
a command and the profile is missing, create it first — a stage that guesses at
`npm test` in a repository that uses something else produces confident garbage.
