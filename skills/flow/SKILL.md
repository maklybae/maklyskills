---
name: flow
description: >-
  The seven-stage development pipeline — explore, plan, implement, test, cleanup,
  review, resolve — plus the task ledger that carries state between stages so no
  work is lost to compaction, a new session, or a subagent boundary. Use this
  skill whenever the user wants a task taken end-to-end rather than a one-off
  edit: when they say "по флоу", "по пайплайну", "прогони весь цикл", "доведи до
  конца", "start a full cycle"; when they ask which stage comes next or want to
  resume a task started earlier; when they hand over a goal and step away
  ("дальше сам", "сделай всё сам", "run it unattended"); or when they hand over
  a ticket or feature big enough that plan-then-build-then-review is the honest
  shape of the work. Also use it to pick which single flow-* stage skill applies
  when only part of the cycle is wanted.
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

The ledger lives outside the repository, under `~/.claude/projects/`. The only
file the pipeline ever writes into the repository itself is the project profile,
because that one is about the repository rather than about you.

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

**Resuming** — read the ledger for the task at
`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/`. Its `stage:`
line says where the work stopped and its findings say what is still open. Do not
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

## Running unattended

Sometimes the user hands over a goal and leaves: "сделай это по флоу, дальше
сам". Two things have to be explicit before that run starts, and they are worth
asking for, because getting them wrong is what makes an unattended run
unrecoverable rather than merely wrong:

- **Where autonomy begins.** Usually straight after the plan is approved — the
  plan is the cheap place to correct course, and everything after it is
  mechanical. Starting fully autonomously from a bare goal is possible, but then
  the plan gets written and followed without anyone reading it, so say that out
  loud rather than letting it happen quietly.
- **Where it ends.** Work left in the tree, committed locally, or pushed with a
  pull request opened. Silence means stop at the tree: an outward-facing action
  does not become authorised by the fact that nobody was watching.

Record both in the ledger header — `mode: unattended, ends at <boundary>`, plus
`budget: N rounds` when the user named one — so a session that resumes the work,
and the user reading it in the morning, know which rules the run was operating
under.

**The three gates do not vanish when nobody can answer. They change shape.**

- **A blocker is fixed or the run stops, and the review loop runs on a round
  budget.** Both rules and the budget's default are `flow-resolve`'s — apply
  them unchanged.
- **A question you cannot ask is not permission to guess quietly.** Do
  everything that does not depend on the answer, take the most conservative
  reading of what remains, and record it as `ASSUMED` — the ledger schema owns
  that protocol, and the closing report leads with those entries.
- **Scope stays where the plan put it.** `flow-implement`'s scope rule, binding
  harder at night: creep is invisible when nobody is watching, so anything
  discovered outside the plan becomes a ledger entry, not an edit.

One practical check before promising to run to the end: an unattended run is
only unattended if the session will not stop to ask permission for each edit and
command. If it will, say so — a run that blocks on the first prompt and then
sits idle is worse than one that never started.

## Project-specific facts

Nothing in these skills hardcodes a build system, a VCS, or a test runner. Those
live in `.claude/flow-profile.md`, written once by `flow-setup`. If a stage needs
a command and the profile is missing, run `flow-setup` first — it owns the
argument for why a guessed command is worse than a missing one.
