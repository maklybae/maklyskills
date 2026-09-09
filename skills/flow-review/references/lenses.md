# Review lenses

A lens is a single question asked of the whole change. Lenses are separate
subagents because a reviewer holding five questions at once answers the easiest
one — usually the one with the most visible surface, which is style.

Each lens below states what it hunts and, just as importantly, **what it must
not report**. Overlap is what produces four copies of the same finding and
makes the merge step guess which one to keep.

## The five defaults

### 1. Correctness and logic

Does the code do what it claims for every input it can actually receive?

Off-by-one and boundary handling; empty, zero, single-element and maximum
inputs; branches that cannot be reached and branches that silently fall
through; error paths that swallow, mask, or return the wrong error; state
machines that can be entered in an order nobody drew; arithmetic that can
overflow, truncate, or lose precision; comparisons on values that are not
comparable the way the code assumes; concurrency — shared state without
synchronisation, check-then-act races, ordering assumed between goroutines or
callbacks.

*Not this lens:* missing tests, deployment behaviour, anything about how the
change interacts with code outside the diff.

### 2. Integration and blast radius

Everything outside the diff that the diff can break.

Callers of every changed signature or behaviour — all of them, found, not
assumed; behaviour that other components depend on and that changed subtly
rather than visibly; data already in production that the new code will read;
schema and migration compatibility in both directions, including a rollback;
serialised formats crossing a version boundary; feature flags and configuration
whose default now means something different; logic that moved between layers,
leaving an orphaned branch behind or losing an error case in transit; contracts
with other services that are now honoured differently.

*Not this lens:* internal logic of the new code itself.

### 3. Production robustness

How this behaves on its worst day, not its intended one.

Calls without a timeout or with one that outlives the request; retries without
backoff, without a cap, or on operations that are not idempotent; work that is
lost, duplicated, or silently dropped when a process restarts mid-flight;
resources that leak under load — connections, goroutines, file handles,
unbounded buffers and queues; partial failure that leaves persistent state
inconsistent because there was no transaction or no compensation; a hot path
that got slower in a way that only shows up at real volume; a failure that will
be invisible in production because nothing logs, counts, or traces it; and the
mirror problem, logging in a hot loop.

*Not this lens:* correctness of the happy path.

### 4. Tests

Not coverage percentage — whether the tests would catch the bug.

Behaviour introduced by this change with no test that would fail if it broke;
error branches and boundaries exercised nowhere; tests that pass regardless of
the implementation because they assert on mocks, logs, or the arguments they
themselves supplied; tests that would break on a rename but not on a defect;
missing regression coverage for a defect the change is meant to fix; fixtures
that hide the interesting case; asynchronous tests that pass by timing luck.

The catalogue of worthless-test forms lives in the `anti-slop-tests` skill. Use
it to *recognise* them rather than re-deriving the list — but report the gap,
not the hygiene. A test that asserts nothing means the behaviour it names is
unprotected, and the unprotected behaviour is the finding; that the test is
badly written is stage 5's business and it has already run. Where the behaviour
turns out to be covered elsewhere, there is nothing here to report at all.

*Not this lens:* whether existing unrelated tests are good — only tests this
change should have brought with it, and the ones it touched. Nor test hygiene
for its own sake: a worthless test is a finding here only when deleting it would
leave a real behaviour with nothing guarding it.

### 5. Security and data handling

Reachable-from-outside failures, judged by what an untrusted caller can do.

Input that reaches a query, a command, a path, a template, or a deserialiser
without parameterisation or validation; authorisation checked at the wrong layer,
for the wrong subject, or not at all on a new entry point; secrets, tokens or
credentials in code, in configuration, or in a log line; personal data written
somewhere it should not persist; error messages that leak internal structure to
a caller who should not see it; a new dependency or endpoint that widens the
attack surface; cryptography or randomness used where the weak variant was
picked by default.

*Not this lens:* general robustness with no adversary in the story.

## Optional lenses

**Design and simplicity** — duplication that wants a shared shape, an
abstraction with one caller, a seam in the wrong place, a type that should exist
but is being passed as three primitives, one module changed for several
unrelated reasons. Add it for large or structural changes. It overlaps
`flow-cleanup`, so running both on a small diff is waste.

**Spec conformance** — requirements in the written spec that are missing,
partial, or implemented differently, and behaviour in the diff that no
requirement asked for. Off by default: specs are frequently the weaker artifact,
and a change that improves on its spec should update the spec, not be reported
as a defect. Turn it on when the spec is the contract — an external API, a
regulated behaviour, a cross-team agreement.

## Choosing the set

Five lenses is the default and fits most changes. Adjust by what the change
actually is, not by its size:

- A schema, migration, or data-format change earns a **bespoke lens** on that
  alone: forward and backward compatibility, what happens to rows written by the
  old code, what a rollback does.
- A worker, queue consumer, or anything concurrent earns a dedicated
  **concurrency** lens split out of correctness.
- A pure refactor with no behaviour change collapses to two: integration/blast
  radius and tests. Correctness of unchanged logic is not in question.
- A change with no untrusted input anywhere near it can drop security — but
  check that assumption before dropping it, since it is the one that is wrong
  most often.

Write a bespoke lens the same way as the five above: what it hunts, and what it
must leave to the others.
