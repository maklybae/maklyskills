---
name: flow-explore
description: >-
  Investigate an implementation task before any code or plan exists — find the
  place in this codebase that already solves a similar problem and should be
  imitated, the constraints that make the task non-trivial, and the prior art
  worth borrowing — and write the result into the task ledger as a brief. Use
  this skill whenever the user hands over a task, ticket or feature and the
  right next move is understanding rather than typing: "разберись", "изучи как
  устроено", "исследуй задачу", "посмотри, как это сделано", "how does X work
  here", or any request to look into something before building it. Prefer it
  over ad-hoc searching whenever the investigation will feed a plan or an
  implementation, so the findings survive as an artifact instead of evaporating
  with the context.
---

# Flow explore

The stage is finished when you can name **the files you would touch** and **the
existing code you would imitate**. Not when you understand the subsystem — you
never fully will, and trying is how exploration runs forever.

## Three sources, in order of value

**1. The codebase's own answer.** Almost every task in a mature repository has a
precedent: a feature with the same shape, one directory over. Finding it is the
single highest-value output of this stage. It settles most design questions for
free, and — more importantly — a change that imitates an accepted pattern is
reviewable in a minute, while an original one makes the reviewer read
everything.

Look for the analogue by shape, not by keyword: another endpoint that does the
same kind of write, another worker with the same failure modes, another client
of the same external system.

**2. The project's history.** Why the current shape is what it is: earlier
attempts, decisions, tickets, ADRs, the ledger of a related task, commit
messages around the code you are about to change. This is the cheapest way to
avoid re-litigating a settled decision or reintroducing something that was
deliberately reverted. If the repository has agent-history search available, a
question like "why was X done this way" is often answered in seconds.

**3. Outside the repository.** Library documentation, a protocol specification,
an established pattern. Worth the detour when the problem is genuinely new
*here* — a library nobody in the repo has used, a protocol with a spec you must
match. It is always subordinate to source 1: an external best practice that
fights local conventions loses, because consistency is worth more than any
individual improvement.

## Search discipline

**Obey the profile's `## Constraints`.** Some repositories punish naive
recursive search badly.

**Locate, then read whole.** Grep hands you a line; the file hands you the
invariants around it. Repeatedly grepping for more fragments of something you
have already located is the most common way to burn context and learn nothing.

**Follow the graph outward from the change point.** Who calls this, who depends
on this behaviour, what persists it. That set is the blast radius the review
will care about later — writing it down now saves the whole loop a round.

**Fan out only when breadth is the problem.** Independent questions across
unrelated parts of a large tree are worth read-only subagents, one question
each. A subagent that comes back with "it's in `store.go`" cost more than
opening `store.go`.

## When to stop

When more searching stops changing the list of files you would touch. Say the
brief out loud to yourself: if you can name the target files, the analogue, and
the constraints, you are done. A thirty-second explore for a task you already
understand is a correct outcome — say so and move on.

## What you produce

Create the ledger if it does not exist — it lives outside the repository at
`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`, schema
in `../flow/references/ledger.md` relative to this skill's directory — and fill
in `## Task`. Then add the brief — in the conversation, and the durable parts in
the ledger:

- **The problem**, in the user's terms, in a few sentences.
- **Where it lands** — files with line references.
- **The analogue** — path, and what specifically to copy from it.
- **Constraints and invariants** that make this non-trivial: concurrency,
  compatibility, data that already exists in production, contracts other
  services rely on.
- **Approaches** — two or three viable ones with their real trade-off. Do not
  pick; `flow-plan` picks, and it will pick better with options on the table.
- **Open questions** — only those whose answer changes the work, each phrased as
  the decision it is, with your recommended answer: the plan gate walks them
  with the user one at a time. Everything else you decide yourself.

## Not in this stage

No code changes, including the "while I'm here" one-line fix. No design decision
committed. No guided tour of the architecture the user did not ask for — the
brief is about this task, and everything else is context they have to pay to
read.
