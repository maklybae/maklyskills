# Slop Test Catalog

The full reference behind `SKILL.md`. Each pattern lists what it looks like, a
before/after where useful, and the bug it fails to catch — because "name the bug
it would miss" is how every one of these is diagnosed.

Examples are mostly Go, since that is where the [Go-specific
section](#8-go-specifics) has published guidance to point at, with Python and
TypeScript where the shape differs. The patterns themselves are
language-agnostic. Read [carve-outs](#carve-outs) before acting — most false
positives come from applying a rule past the context where it holds.

## Contents

- [1. Mock echo and interaction assertions](#1-mock-echo-and-interaction-assertions)
- [2. Tautologies](#2-tautologies)
- [3. Incidental output](#3-incidental-output)
- [4. Assertion sprawl](#4-assertion-sprawl)
- [5. Table-test padding](#5-table-test-padding)
- [6. Redundant scope](#6-redundant-scope)
- [7. Empty and no-assertion tests](#7-empty-and-no-assertion-tests)
- [8. Go specifics](#8-go-specifics)
- [9. Flag, do not silently edit](#9-flag-do-not-silently-edit)
- [Carve-outs](#carve-outs)

---

## 1. Mock echo and interaction assertions

The single largest category in generated suites, and the one with the widest gap
between how thorough it looks and how much it proves.

**1.1 Mock echo.** A double is configured to return a value; the test asserts
that value came back.
```go
repo.EXPECT().GetUser(gomock.Any(), "u1").Return(&models.User{Name: "Alice"}, nil)

got, err := manager.GetUser(ctx, "u1")

require.NoError(t, err)
require.Equal(t, "Alice", got.Name)
```
Delete. The manager could `return repo.GetUser(...)` verbatim, or apply the
wrong tax rate, or drop half the struct — as long as `Name` survives, this stays
green. It proves that gomock works.

The rewrite, when the manager *does* something worth protecting, asserts the
transformation rather than the passthrough:
```go
repo.EXPECT().GetUser(gomock.Any(), "u1").Return(&models.User{DeletedAt: past}, nil)

_, err := manager.GetUser(ctx, "u1")

require.ErrorIs(t, err, models.ErrNotFound)
```

**1.2 Interaction assertions as the whole test.** The test verifies that a call
happened and asserts nothing about the result.
```go
mockClient.AssertExpectations(t)          // the only assertion in the test
```
```python
mock_repo.save.assert_called_once_with(user)   # ...and nothing else
```
```ts
expect(sendEmail).toHaveBeenCalledWith("a@b.c");
```
Delete, or rewrite to assert state. Interaction assertions check *how* the
system reached its result when what matters is *what* the result is — they pin
the call graph, so any refactor that reorganises the calls without changing
behaviour turns them red. That is the definition of a change-detector test.

The exception that is not slop: the interaction *is* the observable behaviour.
"An audit record is written", "the payment gateway is called exactly once on
retry", "nothing is published when validation fails" — for these the call is the
contract, and asserting it is correct. The distinction is whether a user of the
system could tell the difference.

**1.3 Verifying the arguments the test itself supplied.**
```go
require.Equal(t, req.ID, capturedReq.ID)
```
The test passed `req` in and asserts it arrived. Delete unless something between
the two is supposed to transform, validate, or enrich it — in which case assert
*that*, not the identity.

---

## 2. Tautologies

Tests whose assertion is guaranteed by the language, the framework, or the line
directly above.

**2.1 Constructor tests.**
```go
func TestNewManager(t *testing.T) {
	m := NewManager(repo, client)
	require.NotNil(t, m)
	require.Equal(t, repo, m.repo)
}
```
Delete. It asserts that assignment assigns. The compiler already knows. A
constructor test earns its place only when the constructor has behaviour worth
protecting — it validates its dependencies and returns an error, it derives a
field, it registers something.

**2.2 Getter and setter tests.** `TestUser_GetName` setting a field and reading
it back. Delete. Getters are exercised as the assert step of tests that mean
something; testing them directly proves nothing beyond field access. The
exception is a getter that computes, defaults, or normalises — that is
behaviour, not access.

**2.3 Testing the language, the standard library, or a dependency.** That
`json.Marshal` marshals, that a `map` stores what was put in it, that the ORM
issues a query. Delete. If a library is genuinely suspect, the test belongs in a
narrow characterisation test that says so in its name, not scattered through the
suite as if it were your code.

**2.4 Assertions that cannot fail.**
```go
require.NotNil(t, err)     // right after a branch that guarantees err != nil
assert.True(t, len(got) >= 0)
```
```python
assert result is not None
assert isinstance(result, dict)
```
Delete. A type-guaranteed or flow-guaranteed assertion is decoration.

---

## 3. Incidental output

Assertions on things that are outputs of the process rather than of the
behaviour, and that no consumer depends on.

**3.1 Log lines.** Capturing the logger and asserting a message was written.
Delete unless the log line is a contract — an audit trail something parses, a
metric derived from it. Ordinary observability logging is not behaviour; it
changes for reasons unrelated to correctness, and a test on it turns every
reword into a red build.

**3.2 Error message wording.**
```go
require.EqualError(t, err, "failed to fetch user: not found")
require.Contains(t, err.Error(), "invalid")
```
Delete or rewrite to `errors.Is` / `errors.As` against a sentinel. String
matching on errors breaks whenever a wrapping layer is added — the canonical
change-detector — while catching none of the bugs that matter, since it says
nothing about *which* error the caller will actually branch on.

**3.3 Formatting and serialisation nobody parses.** Asserting the exact bytes of
a `String()` implementation, the field order of a JSON blob, or the rendering of
a debug dump. Delete. Assert a round trip instead — marshal, unmarshal, compare
the value — which is the property that actually has to hold.

**3.4 Results that depend on another package's stability.** Assertions pinned to
the exact text another library produces. Even when correct today, that text is
not part of any contract you control.

---

## 4. Assertion sprawl

**4.1 Per-field assertion runs.**
```go
require.Equal(t, "u1", got.ID)
require.Equal(t, "Alice", got.Name)
require.Equal(t, 30, got.Age)
require.Equal(t, "a@b.c", got.Email)
require.True(t, got.Active)
// ...ten more
```
```go
want := models.User{ID: "u1", Name: "Alice", Age: 30, Email: "a@b.c", Active: true}
require.Empty(t, cmp.Diff(want, got))
```
Rewrite to one structural comparison with a diff. The run is not more thorough —
it is the same fact reported from fifteen places, it stops at the first failure
so you learn one field per run, and it silently ignores every field somebody adds
later. A whole-structure compare catches the new field for free.

**4.2 Assertion roulette.** Many bare assertions in one test with nothing to
identify which failed. The fix is usually not messages on each — it is splitting
the test, because a test needing five unrelated assertions is testing five
behaviours. One test, one behaviour.

**4.3 Duplicate asserts.** The same value checked twice in one test, sometimes in
two spellings (`NotNil` then `Equal`). Keep the strongest one.

**4.4 Magic numbers.** Literals with no name and no derivation, so a failure
tells you `42 != 43` and nothing about which rule was violated. Near-universal in
generated suites. Either name the constant or, better, derive the expectation
from the input so the relationship is visible in the test.

---

## 5. Table-test padding

**5.1 Rows that walk the same branch.** Five rows differing only in which valid
string goes in. One case, not five. Keep one representative row per branch, plus
genuine boundaries — the row that is empty, the row at the limit, the row one
past it.

**5.2 Rows that duplicate the implementation's arithmetic.** A `want` computed in
the test by the same expression the code uses. Green whatever both do. Write the
expected value out literally.

**5.3 A table where the cases have different logic.** When rows need conditionals
in the loop body — `if tc.wantErr { ... } else { ... }` growing a third arm —
the shape has stopped fitting. Split into separate test functions. A table is for
one logic path over many inputs; different logic belongs in different functions,
and the branching loop body is itself untested code.

**5.4 Unnamed or unreadable cases.** Rows without a descriptive name, so a
failure reports `#3`. Not deletion-worthy on its own, but fix the name while you
are in there: the name is what makes a failure diagnosable without opening the
file.

---

## 6. Redundant scope

**6.1 Internal tests duplicating public ones.** A private helper tested directly
while the public function that calls it is already covered for the same
behaviour. Delete the inner test. Testing through the public API is what makes a
suite survive refactoring: if such a test breaks, a real caller would have broken
too, and that is the only signal worth having.

The counter-case is real: a private function with combinatorics that would take
twenty public-API tests to reach. Keep that one — but keep it because reaching
the behaviour through the front door is impractical, not by default.

**6.2 The same behaviour tested at three levels.** Unit, integration, and
end-to-end all asserting the same rule. Keep it at the level where the failure
would be most diagnosable, usually the innermost one that covers it, and let the
outer levels test what only they can — wiring, transport, persistence.

**6.3 Tests of code the change orphaned.** Tests for functions that no longer
exist in any meaningful path. They compile, so nothing complains.

---

## 7. Empty and no-assertion tests

**7.1 No assertion at all.**
```go
func TestProcess(t *testing.T) {
	p := NewProcessor()
	p.Process(ctx, input)      // no assertion; passes unless it panics
}
```
Delete, or rewrite to assert the effect. As written it is a smoke test wearing a
behaviour test's name, and its only contribution is to the coverage percentage.

If a panic really is what you are guarding against, say so in the name
(`TestProcess_DoesNotPanicOnEmptyInput`) so the next reader does not mistake it
for coverage of `Process`.

**7.2 Error-swallowing tests.** A body wrapped in `try/except: pass`, or a Go
test that captures `err` and never checks it. Green regardless. Delete or fix.

**7.3 Permanently skipped tests.** `t.Skip`, `@pytest.mark.skip`, `it.skip`,
commented-out bodies. Do not delete silently — [flag](#9-flag-do-not-silently-edit)
them. Somebody disabled the test for a reason, and that reason is information.

---

## 8. Go specifics

Guidance here follows [go.dev/wiki/TestComments](https://go.dev/wiki/TestComments),
the Go project's own test-review conventions. Where it conflicts with the
repository's established style, the repository wins — see
[carve-outs](#carve-outs).

**8.1 Compare full structures, not fields.** Use `cmp.Diff` (with
`cmpopts.IgnoreFields` / `protocmp.Transform` where needed) over a run of
per-field equality checks, and over `reflect.DeepEqual`, which is sensitive to
representation details and prints an unreadable failure.

**8.2 Test error semantics, not error strings.** `errors.Is` against a sentinel,
`errors.As` into a typed error, or a status-code check. Never `err.Error() ==`,
`EqualError`, or `Contains(err.Error(), ...)` — a wrap anywhere upstream breaks
them, and they never distinguished the errors that matter anyway.

**8.3 Got before want, and identify the input.** `got = %v, want %v`, with the
function name and the input in the message or the subtest name. A failure should
be diagnosable from the CI log alone.

**8.4 `t.Error` over `t.Fatal` for ordinary assertions.** `t.Fatal` stops at the
first problem, so a run reports one failure per iteration instead of all of them.
Reserve it for preconditions that make the rest of the test meaningless.

**8.5 `t.Helper()` in helpers.** Without it, failures are reported at the line
inside the helper, and every failure in the file points at the same place.

**8.6 Table-driven vs separate functions.** Same logic, many inputs → table.
Different logic → separate functions. See [5.3](#5-table-test-padding).

**8.7 Subtests get readable names.** They are escaped for the `-run` flag, so
names full of spaces and slashes are painful to target. Descriptive and terse.

**8.8 `time.Sleep` as synchronisation.** A test that passes because a sleep was
long enough is a flake waiting for a loaded CI machine. Synchronise on a channel,
a `sync.WaitGroup`, or a polled condition with a deadline. Flag it rather than
guessing at the right mechanism.

**8.9 Missing the race detector.** Concurrency tests that never run under `-race`
prove almost nothing about concurrency. This is a profile/CI concern rather than
a per-test edit — note it and move on.

---

## 9. Flag, do not silently edit

These are findings for a human. Editing them is how a cleanup pass hides a
defect.

**9.1 A coverage gap the deletion exposed.** Deleting the fake test made the
silence visible. Report the branch that is now uncovered; writing the real test
is the implementation stage's job.

**9.2 A green test asserting wrong behaviour.** The assertion encodes a bug and
the code obliges. Never bring the assertion in line with the code — that
converts a caught bug into a shipped one. Report both.

**9.3 A flaky test.** Passing sometimes, or only in a given order, or only after
a sleep. An intermittent failure is a finding, not noise; concurrency bugs
present exactly this way first.

**9.4 A test that only passes in isolation, or only in a suite.** Shared state
between tests. Report it — the fix usually changes production code or fixtures.

**9.5 Skipped and commented-out tests.** See [7.3](#7-empty-and-no-assertion-tests).

**9.6 Anything you cannot verify locally.** If you cannot run it, you cannot
judge whether deleting it loses coverage. Report it instead.

---

## Carve-outs

The cases where the "slop" reading is wrong. The asymmetry that governs all of
them: deleting a worthless test costs a `git show`; deleting the last test of a
branch costs a production bug.

- **The repository's testing style wins.** A codebase built on `testify`,
  `ginkgo`, a suite type, or a fixture package keeps using it. [8](#8-go-specifics)
  is about what is asserted, not which package spells the assertion. Rewriting a
  suite into a foreign style is not cleanup, it is a rewrite nobody asked for.

- **Regression tests are permanent.** A test naming a bug, a ticket, or a
  specific past failure earns its place forever, however small it looks. The
  triviality is the point — the bug was trivial too, and it shipped.

- **Interaction assertions where the interaction is the contract.** An audit
  record written, a webhook fired exactly once, nothing published on validation
  failure. Asserting the call is correct when a user of the system could tell the
  difference.

- **Tests DAMP, not DRY.** Readable duplication in tests is deliberate: a test
  should be understandable standing alone, without chasing a helper three files
  away. Do not consolidate clear tests into a clever fixture.

- **A large arrange block.** Judge the assertions, not the setup. Complex setup
  often means the test reaches a genuinely awkward state, which is where the
  interesting bugs live.

- **Slow tests.** Integration and end-to-end tests are expensive by nature. Cost
  is not the criterion; whether a plausible bug turns them red is.

- **A test for behaviour that looks obvious.** "It returns an error on empty
  input" reads trivially and is exactly the branch nobody exercises by hand.
  Boundaries and error paths are the highest-value tests in any suite and the
  easiest to mistake for filler.

- **A characterisation test around legacy code.** Deliberately pinned to current
  behaviour, warts included, to make a refactor safe. It looks like a
  change-detector test and it is one — on purpose, temporarily. Check for a name
  or a note saying so before deleting.

On tests, as on code: when in doubt, do not cut. Rewrite instead, and say what
you were unsure about.
