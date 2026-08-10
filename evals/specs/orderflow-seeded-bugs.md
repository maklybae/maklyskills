# orderflow seeded bugs — review/resolve eval material

## Purpose

Measure what `flow-review` actually catches and what `flow-resolve` decides,
against known ground truth: a realistic feature branch with planted defects,
decoys that punish over-reporting, and a clean control branch that measures the
false-positive floor. Built on top of the orderflow fixture
(`evals/specs/orderflow-fixture.md`); read that spec first.

## The carrier: a refund feature branch

Planted bugs do not live in synthetic diffs — they hide inside a genuinely
useful change, the way real bugs do. The carrier is **`add refund flow`** in
`services/orders`: the natural next feature after cancel, with the cancel flow
as its obvious analogue (endpoint + core method + store update + inventory
interaction + tests). The branch must be a competent, mostly-good
implementation — reviewable in earnest, not a strawman.

Deliverable layout (inside the fixture bundle):

```
evals/fixtures/orderflow/
  branches/refund-bugs/steps/     # same step format as the trunk, applied on
                                  # top of HEAD onto branch feature/refund
  branches/refund-clean/steps/    # the control branch, see below
  ground-truth/refund-bugs.json   # the answer key — never enters the repo tree
  witnesses/                      # one witness test per planted bug (see below)
```

`generate.sh` gains a flag: `generate.sh <target-dir> --branch refund-bugs`
materializes trunk + that branch, checked out, with `main` as the merge base.
Without the flag, behaviour is unchanged.

## What gets planted

Target: **7-9 real defects + 4-6 decoys**, distributed across the five review
lenses so every lens has something to find and something to resist.

Real defects (each maps to one lens, severity per the reviewer ladder):

- **correctness (2-3)**: e.g. a status-transition check that admits one wrong
  state; an amount computation that loses cents on partial refunds; an
  idempotency gap the cancel analogue handles but refund does not.
- **integration / blast radius (1-2)**: e.g. refund reuses a store method and
  silently changes an invariant its other caller relies on; a JSON field
  renamed in a struct that the file backend already persisted.
- **production robustness (1-2)**: e.g. the inventory call in the refund path
  lacks the timeout the cancel path has; a retry without idempotency.
- **tests (1)**: a planted test that asserts what its own fake returns —
  green regardless of the implementation.
- **security / data handling (1)**: e.g. refund reason echoed into a log or
  response unsanitized, or an endpoint that skips the validation its peers do.

Hard constraints on every planted defect:

1. **The existing suite stays green on the branch.** A bug the tests already
   catch tests `make test`, not the reviewer. This inverts the SWE-smith gate.
2. **A witness proves it is real.** For each bug, `witnesses/` holds a test
   (or a short script) that is red on the bug branch and green on the clean
   branch. Witnesses never enter the repo tree; the grader and the planting
   validation run them by copying into a scratch checkout.
3. **Reachability.** The trigger is an input, sequence, or state the system
   can actually reach — the reviewer prompt demands a trigger, so every
   planted bug must have one. No "theoretically wrong" plants.

Decoys (each marked `decoy: true` in ground truth; any finding on them is a
false positive):

- code that looks race-prone but is guarded by a lock taken one layer up;
- a missing-looking validation that the type or the single caller guarantees;
- an "inefficient" loop that is correct and irrelevant at real sizes;
- a deviation from the cancel analogue that is deliberate and better;
- optionally one procedural mutation (SWE-smith style) that provably changes
  nothing observable.

## The control branch

`refund-clean`: the same refund feature, same files, same shape and size, with
all nine defects fixed and the decoys intact. Reviewing it measures the
false-positive floor — every finding against it is noise by construction
(caveat: if a reviewer finds a *real* pre-existing issue in trunk code, that is
a fixture finding, not an FP; the grader routes it to triage instead of
counting it). The two branches must be close enough that diff size or file
list cannot leak which branch is under review.

## Ground truth schema

`ground-truth/refund-bugs.json`, one record per plant (Qodo-style):

```json
{
  "id": "B3",
  "kind": "bug | decoy",
  "lens": "correctness | integration | robustness | tests | security",
  "severity": "blocker | serious | minor",
  "title": "refund admits shipped orders",
  "description": "one paragraph: what is wrong, the trigger, the consequence",
  "file_path": "services/orders/internal/core/refund.go",
  "start_line": 41,
  "end_line": 48,
  "snippet": "the exact planted lines",
  "witness": "witnesses/b3_refund_shipped_test.go",
  "resolve_expectation": "fix | accept | reject"
}
```

`resolve_expectation` is what a correct `flow-resolve` verdict looks like when
the finding is reported accurately. Decoys carry `resolve_expectation` from
the reviewer's side: if reported, the correct resolve verdict is `reject`
(code disproves it) or `accept` (real but consciously fine) — state which.

## Grading

- **Hit rule (Qodo)**: a reviewer finding matches a plant only if the
  description identifies the same underlying defect AND the location overlaps
  `[start_line, end_line]` of the right file. Matching judged by an LLM judge
  using the Martian benchmark's published matching prompt as the base;
  location checked deterministically.
- **Metrics**: recall overall and per lens; precision over all findings;
  FP count on decoys (named separately — decoy hits are the expensive kind of
  noise); FP floor from the clean branch; rounds to convergence when the full
  review/resolve loop runs.
- **Resolve scoring**: given the bug branch's findings, `flow-resolve` is
  graded on verdict agreement with `resolve_expectation`, with reasons
  present. A silently-dropped finding is a fail (silence is not a verdict).

## Validation gates before first use (the planter runs these)

1. `make lint && make test` green on both branches at their tips.
2. Every witness red on `refund-bugs`, green on `refund-clean`.
3. Determinism: double materialization of both branches yields identical tip
   SHAs.
4. No ground-truth vocabulary in either branch (tree and messages): no bug
   ids, no "planted", "decoy", "witness", plus the fixture's existing leak
   list.
5. Branch history looks like feature work (3-5 commits, plausible messages,
   one author or two), not like a dump.
6. Self-check: for each plant, re-read the surrounding code and confirm no
   existing guard silently neutralizes it — a plant that cannot fire is a
   decoy mislabelled as a bug and will corrupt recall.

## Acceptance regime v2 — the known control

Adopted after three fix-breeds-bugs rounds proved a "clean" control has no fixed
point. The control arm is **known, not clean**:

- Every ground-truth record carries an `arms` field (`["refund-bugs"]`,
  `["refund-clean"]`, or both). A real defect found in refund-clean is admitted
  by writing a C-record plus a witness — it does not have to be fixed.
- **Witness parity is the release gate**: every `bug` record has a witness
  (decoys prove themselves by being un-triggerable and carry none); every
  bugs-arm witness is red on refund-bugs and green on refund-clean; no witness
  is red on refund-clean. A witness whose mechanism does not exist on an arm
  (e.g. the sidecar on refund-bugs) declares an explicit skip that counts as
  green there. A witness red on clean means: fix the defect or re-arm the
  record — the gate stays deterministic either way.
- **Grader**: a finding on refund-clean matching a recorded C-record is a hit,
  not a false positive. The FP floor is findings that match nothing recorded;
  decoy hits are counted separately.
- **Shape freeze**: identical file lists across arms (hard gate); per-file line
  counts within ~10% for files where both arms implement the same mechanism;
  files that cannot meet that by construction (the clean arm carries coverage
  and mechanisms the bugs arm must lack) live in a declared exceptions table
  with a cap and a one-line reason each. A fix that widens a file's spread is
  rejected in favour of recording the defect.
- **Fix budget**: at most two hardening rounds per arm, ever. Afterwards every
  adjudicated finding lands as ground truth (bug with `arms`, or decoy) and
  nothing is edited.
- **Adjudication ledger**: `ground-truth/adjudications.md` records every claim →
  verdict → deciding evidence, so intended/wrong calls are never re-litigated.
- Review-based checks, if used at all, run with a reviewer configuration
  different from the one under evaluation, over both arms in one batch, and
  gate on the adjudicated delta — never on a raw "zero findings on clean".

## Out of scope here

- Scenario prompts, harness wiring, and grader implementation — separate
  deliverable (the harness spec).
- Planting bugs in trunk history ("archaeology" evals) — later, if ever.
