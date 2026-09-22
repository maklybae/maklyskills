---
name: flow-implement
description: >-
  Build an approved plan stage by stage, writing code that looks like the code
  already in the repository, honouring its rule files, testing each stage as it
  lands, and verifying before moving on. Use this skill when the user says
  "реализуй", "имплементируй", "погнали делать", "implement the plan", "давай
  кодить", when an approved plan or openspec change exists and the next move is
  building it, or when a task is small enough to skip planning but still needs
  code written in the local idiom rather than a generic style. Prefer it over
  free-form editing whenever the change spans several files or a plan already
  says what to build.
---

# Flow implement

## Before writing anything

Read, in this order: the ledger
(`~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`,
schema in `../flow/references/ledger.md` relative to this skill's directory),
the plan (or openspec change; when none exists, see below), the profile, the
repository's rule files, and —
completely — the file the current stage names as its **Model**. Skimming the model defeats the point: you are about to imitate
it, and the parts worth imitating are the error handling and the invariants, not
the shape you can guess from the signature.

## When there is no plan

On the mechanical and contained routes there is no plan file. Small does not
suspend the loop — it collapses the plan to a single stage you write yourself
before touching code: the files you will change, the model file to imitate, the
verify command from the profile. Say those three lines to the user first, right
after the route line; that is the plan gate, at the price this task deserves.
On the mechanical route the two collapse into one — the files are the ones the
parameter passes through, and the model is the neighbour that already does the
same thing. If you cannot fill the three slots without going searching, the
task was not small — move the route up and run flow-explore and flow-plan
instead of discovering that mid-edit.

The ledger follows the same economy. The umbrella skill's sizing rule says when
it exists: never on the mechanical route, from the first review round on the
contained route, from explore on the feature route. Create it earlier only when
the work produces something a later session will need — a decision worth
recording, a finding — and say so.

## The per-stage loop

1. Read the model file.
2. Write the code in its idiom.
3. Write or extend the tests **in the same stage**, for the behaviour that stage
   introduces.
4. Run that stage's `Verify` commands.
5. Update the ledger: stage done, and any decision you made that the plan did
   not anticipate.

Do not start the next stage while the current one is red. A broken stage under a
finished one costs far more to untangle than to fix now, because by then you no
longer know which of the two changes caused it.

## Imitate, don't improve

The local pattern beats the abstract best practice, every time. A change that
looks like its neighbours is reviewable at a glance; one that imports a foreign
style forces the reviewer to re-derive the whole file. Naming, error wrapping,
layering, how dependencies get injected, how queries are written, how tests are
organised — copy what is there.

The one exception is a local bug. If the model file does something genuinely
wrong, do not propagate it and do not silently fix the neighbour either: record
it in the ledger as a finding and say so. Fixing unrelated code inside a feature
change is how a reviewable diff becomes an unreviewable one.

## Write it the way it will survive cleanup

`flow-cleanup` strips comments that restate the code, doc-headers that re-type
a signature, and tests that assert a mock returned what it was told to return.
Writing them now only to delete them in an hour is wasted work on both ends.

So: no comment that a better name would make unnecessary — try the name first,
and the need usually disappears. No doc-header on a symbol just because it is
exported. The full doctrine lives in the `anti-slop-code` skill; the short
version is that every line you add should be one a reviewer would miss if it
were gone.

The same economy governs step 3 of the loop. Write the test for the behaviour
the stage introduces, then ask what bug it would catch — if you cannot name one,
you have written a test that pins the implementation rather than protecting it,
and cleanup will delete it. The failure mode to avoid is writing one test per
exported symbol: that is coverage of the *surface*, and it is how a change
arrives with thirty tests and two of them load-bearing. Cover the branches that
can actually go wrong — errors, boundaries, round trips — and let the rest go
untested rather than tested emptily. `anti-slop-tests` holds the full catalogue
of what not to write.

## Scope

Build the current stage and nothing else. Noticing that something nearby is
broken, ugly, or missing is a *finding* — write it in the ledger, mention it,
keep going. Unrequested improvements are how a two-file change becomes a
twelve-file change that nobody can review.

If the stage cannot be built as written, the plan was wrong — stop and say what
you hit, then amend the plan. Silently improvising a different design is what
makes a review report "the code does not match the plan" three stages later,
when the divergence has already spread.

## Generated code

Regenerate it; never hand-edit it. When a stage changes an interface, a proto,
or a schema, regenerating the mocks and stubs is part of *that* stage, not a
cleanup task for later — the profile's `## Commands` records how.
