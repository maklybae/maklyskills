---
name: flow-plan
description: >-
  Turn an explored task into a staged implementation plan whose every stage
  names real symbols, real files, the existing code to imitate, and the exact
  command that proves it is done — writing it as an openspec change when the
  target service uses openspec, and as a plan file otherwise. Use this skill
  when the user asks for a plan, says "составь план", "распиши этапы", "как
  будем делать", "plan this out", when a task is big enough that implementing it
  straight would mean improvising design decisions mid-edit, or right after
  flow-explore has produced a brief. Prefer it over jumping into edits whenever
  the change spans more than one file or introduces a new concept.
---

# Flow plan

A plan exists to make the next stage mechanical. If implementing still requires
deciding what to build, the plan has not done its job.

## Where the plan lives

**If the target service uses openspec, the plan *is* the openspec change.**
Detect it by walking up from the directory you are changing until you find an
`openspec/` directory — in a monorepo these sit per service, not once at the
root, so the nearest one is the right one. The profile's `## Spec workflow`
section records the roots.

Then use the repository's own openspec tooling — the `openspec` CLI, or the
openspec skills the project ships — to create the change and its artifacts
(proposal, design, tasks), and validate it. Record the change id in the ledger.
Do not also write a separate plan file: two plans drift apart within a day, and
nobody knows which one the code follows.

**Otherwise**, write `plan.md` next to the ledger, in the same directory.

Either way the ledger — outside the repository at
`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`, schema
in `../flow/references/ledger.md` relative to this skill's directory — points at
the result through its `spec:` line, so later stages and review subagents can
find it.

## The unit of a plan is a verifiable stage

```markdown
### Stage 3: Delete files by id list

**Do:** add `DeleteByIDList(ctx, ids []string) error` to `files.Repository` and
to both the SQL and in-memory implementations; soft-delete by setting
`DeletedAt`, keep the rows.
**Files:** `internal/repositories/files/repository.go` — interface;
`.../sql.go`, `.../mem.go` — implementations.
**Model:** `internal/repositories/workflows/sql.go` — batched delete with named
parameters, same shape.
**Verify:** `go test ./internal/repositories/files/...`
```

What makes stages like that work:

**Real names.** "Add methods to the repository" is a wish. The version above is
a plan, and the difference is that a reader can tell whether it was done.

**One action per item.** "Create or extend the manager" means the plan was
written without looking. Go look, then write the one that is true.

**Every stage ends in a command.** A stage with no verification is a stage whose
completion is a matter of opinion. If no command can prove it, the stage is
described at the wrong altitude — split it until one can.

**A stage that changes a shared surface verifies every consumer.** Building the
one binary you are working in proves nothing about the others that compile the
same symbol — find who else consumes the changed constructor, function, or type,
and put a build that covers all of them on that stage's Verify line. "Build Ok"
scoped to one consumer is the exact shape of a miss that surfaces two stages
later as someone else's compile error.

**Tests belong to the stage that adds the logic.** A final "write the tests"
stage is where coverage goes to die: by then the deadline is closer, the
behaviour is fuzzy, and the tests get written to match the code rather than the
requirement.

**Regeneration is its own step.** Changing an interface, a proto, or a schema
means regenerating mocks, stubs, or migrations. It is the most commonly
forgotten step in any plan, and it fails loudly three stages later.

**Stages depend only on earlier stages.** If stage 4 needs something from stage
6, the order is wrong.

**Sizing.** If you cannot say in one sentence what "done" looks like, or the
stage touches more files than you can hold in your head, split it.

## Name the risky part

Somewhere in the plan there is a stage that might be wrong — an assumption about
how an external system behaves, a schema you have not seen with real data. Say
which one it is, and put the cheap experiment that would disprove it *before*
the expensive stage that depends on it. A plan that hides its uncertainty is
optimistic fiction.

## The gate

Present the plan and stop. This is the one blocking question in the pipeline
that always pays for itself: changing direction here costs a paragraph, and
changing it after implementation costs the implementation.

Present it compactly — the stage titles and what each produces — and offer the
detail rather than dumping it. If the user says to go ahead without reading, go
ahead; it is their call and their time.

**When the plan has real forks, the gate becomes an interview.** One obvious
approach and no risky stage — present compactly, one exchange, as above. Real
trade-offs — walk them with the user one at a time, in dependency order, each
with your recommended answer and its reason: several questions at once are
bewildering, and a question without a recommendation outsources the thinking.
Two rules keep it honest. A fact is never a question — if the repository, the
history or the docs can answer it, exploration left a gap, so go look. And only
decisions that change the work earn a question. Unattended runs cannot
interview; the ledger's ASSUMED protocol owns that case.
