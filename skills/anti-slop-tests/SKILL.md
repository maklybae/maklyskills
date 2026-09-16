---
name: anti-slop-tests
description: >-
  Detect and delete "AI slop" from test suites — the change-detector tests,
  mock-echo assertions, per-field assertion runs, duplicate table rows and
  coverage-driven filler that language models emit by the dozen. Enforces one
  rule of value: a test earns its place only if it would fail when a plausible
  bug is introduced into the behaviour it names, proven by an actual
  falsification probe when the case is arguable. Use this skill whenever the
  user asks to clean up, prune, thin out or de-slop tests; whenever they say the
  tests look "AI-generated", "тупые", "бесполезные", "их слишком много",
  "выкинь лишние тесты", "почисти тесты", "prune these tests", "which of these
  tests are worthless"; whenever a change has just arrived with a large batch of
  generated tests; and as the tests pass of a pre-review cleanup. Prefer it over
  an ad-hoc tidy so the value rule and the delete-vs-rewrite split are applied
  the same way every time.
---

# Anti-Slop Tests

## What a slop test is (and why it survives review)

A slop test is one that passes whether or not the code is correct. It asserts
that a mock returned what it was told to return, that a constructor assigned the
fields it was handed, that a getter gets. Google's name for the general shape is
the **change-detector test**: it fails when the code *changes*, not when the code
*breaks*. It reports churn and calls it coverage.

Three properties make this worth its own pass rather than a note in a review:

- **It is invisible to the usual defences.** It is green, it is formatted, it
  raises the coverage number, and it is written in the same style as the tests
  that do work. Nothing in CI distinguishes it.
- **It is worse than no test.** A missing test is a known gap. A slop test is a
  gap that reports itself as covered, and it charges rent: every refactor must
  update it, so it makes the code *harder* to change while protecting nothing.
- **Language models produce it structurally, not accidentally.** Given "write
  tests", a model optimises for plausible-looking coverage of the surface it can
  see — every public method, every field, every branch it can name — because
  that is what the training distribution rewards. Empirical studies of
  LLM-generated suites find magic-number tests approaching 100% prevalence and
  assertion roulette in over half of zero-shot output. The volume is the tell:
  a human writing tests by hand stops when the risk is covered; a model stops
  when the surface is.

Cleaning this is restoring the suite to what an experienced engineer who already
knows this codebase would have left behind: fewer tests, each one load-bearing.

## The rule of value

> **A test earns its place if it would fail when a plausible bug is introduced
> into the behaviour it names.**

Everything below is an application of that one sentence. Apply it by **naming
the bug out loud**: state the specific defect this test is standing guard
against. If you cannot name a defect that would turn it red, it is not testing
anything — it is describing the implementation back to itself.

Two corollaries worth stating separately, because they catch the cases the
headline rule reads past:

- **Substitution.** If the implementation were thrown away and rewritten from
  scratch with identical observable behaviour, this test should still pass. One
  that would not is testing the implementation, not the behaviour.
- **The name is part of the claim.** A test named for a behaviour it does not
  actually exercise is a false report, and it is the reason the gap stays
  invisible. When you keep such a test, fix the name.

## Settle it by reading first

Trace where each asserted value comes from. If every one of them originates in a
stub the test itself configured, no mutation of the production code can reach
it: the test is green under every mutant, and that is proven by the data path,
not guessed. The same reading disposes of a test whose assertions never touch
the changed code at all.

Reading costs nothing; the probe below costs an incremental rebuild of the
package. Spend it only where the data path is genuinely unclear — and notice the
pull in the other direction, because a measurement looks more rigorous than an
argument even when the argument is already conclusive.

## The falsification probe

The rule of value is a thought experiment, and the agent running it is usually
the one that wrote the tests. Where reading leaves the case open, do not reason
harder — **measure**:

1. Introduce the named bug into the production code: invert one condition, drop
   one guard, return the zero value, skip one write.
2. Run **only** the test under suspicion.
3. Green under the mutation means the test does not protect what its name
   claims. That is the verdict, and it is not a matter of opinion.
4. **Revert the mutation immediately**, before anything else — including before
   writing up the result. Then confirm the tree is clean against the diff
   command before continuing.

A test that asserts a mock returned its own stub value needs no experiment — the
section above already settled it. A test that looks pointless but sits on a real
error branch is what the probe is for, and it is exactly where guessing is
expensive.

Never batch mutations, and never leave one in the tree while you go read
something else. The failure mode this rule prevents is a cleanup pass that ships
a deliberately broken line.

## Workflow

1. **Scope.** The diff, named files, or a package. When cleaning up after recent
   work, prefer the diff — judge the tests this change brought, not the whole
   suite. Tests that predate the change are someone else's decision and often
   encode a regression you cannot see.

2. **Pre-scan.** Cheap high-recall candidates before reading:
   ```
   python3 scripts/scan_tests.py <files-or-dir>
   git diff --name-only | python3 scripts/scan_tests.py --stdin-list
   ```
   Candidates, never verdicts. The scanner cannot see whether a test is the last
   thing covering a branch, which is the one fact that overrides everything else.

3. **Read and judge.** For each test, name the bug it guards against. Cross-check
   against `references/patterns.md` for the concrete forms and the carve-outs.
   Assign one of four verdicts (below).

4. **Trace the data path.** Probe only what reading leaves open.

5. **Apply.** Delete, rewrite, or leave. Then run the suite: the pass must end
   green, and a rewrite is exactly as capable of breaking as a code edit.

6. **Report.** Inline, in the format below. Do not write a report file unless
   asked.

## The four verdicts

**DELETE** — the behaviour is worthless to protect, or already protected
elsewhere through the public entry point. The catalogue's whole first half lands
here.

**REWRITE** — the test is badly written but is the only thing standing between a
real branch and silence. This is the verdict that keeps the pass honest, and the
one most easily skipped. Deletion is correct when coverage is duplicated or was
never warranted; it is *not* correct when it is the last coverage of a branch,
however poorly that coverage is expressed. Rewrite it to assert the observable
result rather than the interaction, and rename it after the behaviour.

**KEEP** — it already earns its place. Say nothing; do not restyle it. A test
that is merely un-idiomatic is not slop.

**FLAG** — a finding for a human, not an edit:
- **A coverage gap this pass exposed.** Deleting the fake test made the silence
  visible; writing the real test is `flow-implement`'s job, not this pass's.
- **A test that reveals a production bug.** It asserts wrong behaviour and is
  green, which means the code has the bug the assertion encodes. Never "fix" it
  by editing the assertion.
- **A skipped, commented-out, or permanently-failing test.** Someone disabled it
  for a reason nobody wrote down.
- **A flaky test.** Timing-dependent, ordering-dependent, or passing only under
  the right sleep. It is a defect report, not a cleanup item, and re-running it
  until green is how a real concurrency bug ships.

## The catalogue (summary)

Full forms, before/after pairs and carve-outs live in `references/patterns.md`.

**Delete on sight:**

- **Mock echo.** A double is told to return `X`; the test asserts `X` came back.
  The only thing proven is that the mocking library works.
- **Interaction assertions as the whole test.** `AssertExpectations`,
  `assert_called_once_with`, `toHaveBeenCalledWith` and nothing else. It checks
  *how* the result was reached, when only *what* the result is matters.
- **Tautologies.** A call verified with the arguments the test itself just
  passed in; a constructor test asserting the fields it handed over; a getter
  test; a test of the standard library or the framework.
- **Assertions on incidental output.** Log wording, error message text, the
  field order of a serialisation nobody parses, `String()` formatting.
- **Per-field assertion runs.** Fifteen assertions walking fifteen fields of one
  struct. Compare the structure once with a diffing comparison — the run is
  fourteen extra failure sites reporting one fact.
- **Duplicate table rows.** Rows that walk the same branch with different
  numbers. Distinct data through one path is one case, not five. Rows differing
  only in a magic number are the machine-generated form of this.
- **Redundant scope.** A test pinned to an internal detail when the same
  behaviour is already covered through the public entry point. Delete the inner
  one; the outer one is the contract.
- **Empty and no-assertion tests.** Calls the code, asserts nothing, passes
  unless something panics. Coverage-number filler.

**Keep and strengthen:**

- Contracts other code depends on — observable behaviour at the public API.
- Error branches, especially awkward ones. Nobody exercises those by hand.
- Boundaries: empty, zero, one, maximum, expired, malformed, concurrent.
- Round trips: persistence, serialisation, compatibility with data already
  written by the old code.
- Regressions with a history. A test that exists because something once broke
  earns its place permanently — make sure its name says what it protects, and
  never delete one because it looks trivial. Triviality is the point: the bug
  was trivial too.

## Clean vs flag

| Delete or rewrite directly | Flag for a human |
|---|---|
| Mock echo, interaction-only assertions | A coverage gap the deletion exposed |
| Tautologies, getter/constructor tests | A green test asserting wrong behaviour |
| Assertions on logs, message wording, formatting | A skipped or commented-out test |
| Per-field assertion runs → one structural compare | A flaky or timing-dependent test |
| Duplicate table rows walking one branch | A test that only passes in a given order |
| Tests of the language, framework, or a library | Anything whose correctness you cannot verify locally |
| Empty and no-assertion tests | |

The asymmetry that governs the split: a wrongly deleted worthless test costs one
`git show`; a wrongly deleted *last* test of a branch costs a production bug. So
**when in doubt about whether a branch is covered elsewhere, rewrite rather than
delete** — and check for other coverage before you decide, rather than assuming.

## Report format

```
Tests: <N> → <M>   Probed: <K> (<J> stayed green under mutation)

## Deleted
- <file>:<line> — <test name> — <what it actually asserted>

## Rewritten
- <file>:<line> — <test name> — <what it now asserts instead>

## Flagged (not changed)
- <file>:<line> — <the concern>, why it matters, suggested direction
```

The count is the one number narration cannot fudge, and on this pass it must go
down or stay flat — a tests pass that ends with more tests has done a different
job than the one it was asked for. But the count is not the goal: **the number
of behaviours actually protected must not drop.** State it plainly if a deletion
lowered it, as a flagged gap rather than a footnote.

If nothing was slop, say so. A clean pass that changes nothing is a valid
outcome; manufacturing deletions to look busy trades a maintenance cost for a
production risk.

## Guardrails against false positives

- **Local convention wins.** If the codebase tests with an assertion library, a
  suite type, a fixture helper or a mocking framework, keep using it. The Go
  wiki's advice against assert libraries is not a licence to rip `testify` out
  of a repository built on it — the target is fifteen assertions where one
  belongs, not the package they are written with. Never import a foreign testing
  style into a repository that has one.
- **A test with a history is not trivial.** Before deleting anything that looks
  too small to matter, check whether it names a bug or a ticket. That is the
  shape of a regression test, and it is permanent.
- **Setup is not assertion.** A test that constructs a lot before acting is not
  slop for that reason; judge the assertions, not the arrange block.
- **Duplication in tests is allowed.** Tests are DAMP, not DRY — a little
  repetition that makes a test readable standing alone is correct. Do not
  refactor three clear tests into one clever helper; that is the opposite of the
  job.
- **No logic in a test is a rule about writing, not a mandate to rewrite.** A
  loop or a conditional in an existing test is a smell worth noting; it is not
  by itself a reason to touch a test that earns its place.
- **Integration and end-to-end tests look wasteful and are not.** Slowness is
  not slop. Judge by whether a plausible bug turns them red.
- **Do not chase the coverage number in either direction.** Coverage is not the
  goal; it never was. Deleting tests to feel lean is the same error as writing
  them to feel covered.
