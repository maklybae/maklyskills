# Slop Pattern Catalog

The full reference behind `SKILL.md`. Each pattern lists what it looks like, a
before/after where useful, and why it is slop. The recurring theme: a slop
pattern is one that adds tokens without adding information, or adds structure
without adding a reason. Read the [carve-outs](#carve-outs) section before acting
— most false positives come from applying a rule past the context where it holds.

Examples span several languages on purpose; the patterns are language-agnostic.
Match the fixed version to the target file's own conventions, not to the language
used in the example here.

## Contents

- [1. Comments](#1-comments)
- [2. Naming](#2-naming)
- [3. Over-abstraction and over-engineering](#3-over-abstraction-and-over-engineering)
- [4. Error handling](#4-error-handling)
- [5. Structure and boilerplate](#5-structure-and-boilerplate)
- [6. Cosmetics](#6-cosmetics)
- [7. Correctness and security — flag, do not silently clean](#7-correctness-and-security--flag-do-not-silently-clean)
- [Carve-outs](#carve-outs)

---

## 1. Comments

The largest category, and the one with the strictest rules. `SKILL.md` holds the
doctrine — zero human-readable comments, directives excepted; one-line
invariants only on paths the project profile allows; no doc-header without the
human asking in this session; move a real why into a name, a test name, docs or
the report before deleting it. This section is the concrete forms.

**1.1 Restating the code.** The comment says what the next line already says.
```
counter++            // increment the counter
user = User()        # create a user
return result        // return the result
```
Delete. The code is the documentation.

**1.2 Translating a name.** A comment that is the function/variable name in prose.
```
// createTexture
createTexture()
// enable extensions
enableExtensions()
```
Delete. If `createTexture` needed explaining, the explanation would be a *why*,
not a restatement of the verb.

**1.3 Generation narration.** The model describing its own output.
```
// Now we loop through the users
// First, we initialize the client
// Here's the helper that formats the date
// As we can see, this returns early
```
Delete. These are conversational artifacts, a dead giveaway of raw LLM output.

**1.4 Docstrings.** Any docstring, trivial or not, public or private.
```python
def add_numbers(a: int, b: int) -> int:
    """Adds two numbers and returns the result."""   # slop
    return a + b
```
Delete the docstring. A codebase that reads like a language tutorial — every
function documented — is a strong slop signal, and a docstring on a private
helper is the same comment with less excuse.

**1.5 Hollow claims and unverified metrics.**
```
// blazing-fast, production-ready implementation
// optimized for performance
// 75% faster than the old version
```
Delete. An adjective like "robust" or "seamless" with nothing behind it is
noise; a number without a benchmark is a guess dressed as a fact. A real
measurement belongs in the benchmark or the PR description, not beside the code
it will stop describing.

**1.6 Section banners.**
```
// ============ INITIALIZATION ============
```
Delete. If a file needs banners to be navigable, it usually needs to be split.

**1.7 Doc-comments on public symbols.** Banned in every language: Go
doc-comments on exported identifiers, Python docstrings on public
classes/functions, JSDoc/TSDoc on `export`ed members, Rust `///` on `pub` items.
```go
// NewPaymentManager creates a new PaymentManager.
func NewPaymentManager() *PaymentManager { ... }
```
```python
class SessionStore:
    """Stores sessions."""          # slop: the class name already said this
```
```ts
/**
 * Formats a date.
 * @param date the date to format
 * @returns the formatted date
 */
export function formatDate(date: Date): string { ... }
```
Delete all three. Each one is the signature retyped in prose, and the `@param`
/ `@returns` scaffolding is pure ceremony where the types already say it.

The only authorization is the human asking for it in this session. A linter rule
is not — when `revive`, `pydocstyle`, or `require-jsdoc` fires on a
self-explanatory name, the linter is wrong — and neither is a package whose
other symbols are already documented.

Where a doc-header carries something real, move it before deleting — see 1.10.

**1.8 Multi-line blocks.** Outside allowed paths a block goes whole, like any
other comment. On a path the profile allows, the one-line cap still holds: two
or more consecutive human-readable lines is a defect even when every line is a
legitimate why.
```python
# This function takes the raw events, groups them by user id,      # slop:
# then counts how many events each user has, and finally returns   # 4-line
# a summary dictionary mapping each user id to their event count   # block
# so the caller can render the per-user report.
def summarize(events): ...
```
```python
def summarize(events): ...          # the name already carries it
```
On an allowed path, keep the load-bearing sentence as one line at the line it
guards and move the rest (1.10).

Directive blocks are exempt and stay byte-for-byte: license/copyright/SPDX
headers, `Code generated by ... DO NOT EDIT.` banners, build tags and pragmas
(`//go:build`, `# type: ignore`, `/* eslint-disable */`).

**1.9 Stateful / historical comments.**
```
// previously used a map here, changed to a slice for NEURO-1234
// fixes the bug from the last PR
```
Delete. Version control holds the history. On an allowed path, the stateless
invariant underneath, if there is one, may survive as one line.

**1.10 Moving a real why.** The move that makes zero safe. Before cutting a
comment that carries a unit, an external contract or a reason, find it a home:

*Into a name or type* — the strongest fix, because it cannot go stale.
```go
// amount is expressed in cents, not whole currency units
func Charge(ctx context.Context, amount int64) error
```
```go
func Charge(ctx context.Context, amountCents int64) error
```

*Into a test name* — for a rule the code enforces. The test that pins the rule
carries it in its name and fails when the rule breaks, which a comment never
does.
```go
// Pay charges the account. It rejects non-positive amounts because the
// upstream gateway silently treats them as full refunds.
func (m *OrderManager) Pay(ctx context.Context, amountCents int64) error {
	if amountCents <= 0 {
		return ErrInvalidAmount
	}
```
```go
func (m *OrderManager) Pay(ctx context.Context, amountCents int64) error {
	if amountCents <= 0 {
		return ErrInvalidAmount
	}
```
```go
func TestPay_RejectsNonPositiveAmount_GatewayTreatsItAsRefund(t *testing.T)
```
Rename a test that already covers the guard. Do not write one during the pass:
a rule nothing tests is a finding for the report, not an edit.

*Out to `docs/`* — for a reason that needs a paragraph, when the repository
already documents this component. No pointer comment is left behind.

*Into the report* — everything else. The *Relocate* section lists the text and
the line it explained, and the author carries it into the commit message or the
PR description, where the why of a change belongs.

Never move a why into behavior: an error string or a log line reworded to hold
the explanation is a behavior change.

**1.11 Trailing comments.** A comment at the end of a code line is still a
comment.
```go
buf := make([]byte, 4096) // one disk page
```
```go
buf := make([]byte, pageSize)
```
Delete, moving the content as in 1.10. Trailing directives — `//nolint`,
`# type: ignore`, `# noqa` — stay.

**Survives:** directives the tooling reads (`//go:build`, `//go:embed`,
`//go:generate`, `# type: ignore`, `# noqa`, `eslint-disable`, `Deprecated:`,
Go example `// Output:`, license and generated-file banners), and one-line,
English, stateless invariants on paths the profile's `## Comments` section
lists — only where deleting them lets a plausible edit break the code silently.
Nothing else.

---

## 2. Naming

Slop naming is about names that carry no domain information — not about the
presence of a role word.

**2.1 Content-free names.** A name that satisfies the syntax and says nothing.
```
data, result, temp, tmp, val, obj, item, info, payload, res, retval, thing
```
Rename to what the value actually is:
```
result  -> parsedInvoice / sortedSessions / validationError
temp    -> formattedDate / normalizedInput
data    -> rawTranscript / userProfile
```

**2.2 Empty qualifiers.** The role word is fine; the qualifier adds nothing.
```
DataManager, InfoHandler, BaseManagerImpl, MyService, ThingProcessor
```
The fix is a *domain* qualifier, not removing the suffix:
```
DataManager -> SessionManager / SubscriptionManager
InfoHandler -> WebhookHandler
```

**2.3 Overlong legalese names.** Rigidly over-applied "readability".
```
total_user_input_character_count   -> char_count
the_list_of_all_active_user_ids    -> active_user_ids
```

**2.4 Inconsistent naming in one scope.** The same concept under three names.
```
userData ... user_data ... data        // one thing, three names
filename ... file_path ... path         // pick one and use it
```
Unify to a single name that matches the file's convention.

**2.5 Domain-dialect mismatch.** Generic terms where the codebase has a
vocabulary. If the code speaks `LedgerEntry` / `AccountId`, do not introduce
`record` / `userId` for the same concept. Conform to the established dialect.

Naming carries extra weight under the comment doctrine: a name is the preferred
home for anything a deleted comment used to say (see 1.10). Renaming a parameter
to `amountCents` retires a comment permanently; a comment saying the same thing
survives only until someone edits the line below it.

See [carve-outs](#carve-outs) for why `OrderManager`, `PaymentClient`,
`AuthHandler`, `i`, `err`, and `ctx` are all fine.

---

## 3. Over-abstraction and over-engineering

Root cause: training on enterprise codebases where patterns appear regardless of
fit. The test for every abstraction: *if you cannot immediately say why it exists,
it probably shouldn't.*

**3.1 Single-use helpers.** A function extracted for one call site that does not
aid readability. Inline it.
```python
def _get_upper(s): return s.upper()   # one caller, no clarity gained
name = _get_upper(raw)                # -> name = raw.upper()
```

**3.2 Speculative layers "just in case".** A 20-line script wrapped in a class
with an abstract base and three helpers; a repository pattern bolted onto direct
data access; a service layer with one method that forwards to another. Collapse
to the level the problem actually needs.

**3.3 Factories/interfaces with one implementation.**
```
UserManagerFactory -> UserManager -> UserRepository   // for one concrete path
```
Collapse the chain. An interface earns its keep when there is a second
implementation (or a real test double the code depends on), not before.

**3.4 Wrong / forced abstraction.** Merging two things that only look alike into
one config-driven mega-component.
```
UserCard + ProductCard  ->  UniversalCard({data, type, isOnline, isOnSale, ...})
                            // body full of `type === 'user' && ...`
```
Prefer two clear components. Duplication is cheaper than the wrong abstraction;
when the two diverge later, the merged one becomes a maintenance trap.

**3.5 Wrapper functions and pointless intermediates.**
```
const result = computeValue();
return result;                 // -> return computeValue();
```
```
addr = addr                    // self-assignment, delete
```
Each unnecessary name is one more thing to track and a place for a bug to hide.

**3.6 Pattern cargo-culting.** Singleton/observer/strategy/factory applied
because they were learned, not because the problem calls for them. "Three classes
and an interface to format a date string." Remove the ceremony.

---

## 4. Error handling

Two opposite failure modes, both slop.

**4.1 Blanket wrapping.** Every operation in its own `try/except Exception`, or a
custom error hierarchy for a one-shot script. Remove the bubble-wrap where a
failure should just propagate; keep handling where a specific recovery exists.

**4.2 Swallowed exceptions (the dangerous one).**
```python
try:
    charge(card)
except Exception:
    pass                          # hides a real failure behind a clean surface
```
```js
try { save(); } catch (e) { /* ignore */ }
```
This is the most dangerous slop because standard review is not calibrated to
catch a silent failure. If the swallow is clearly stylistic (e.g. suppressing a
cleanup error that truly doesn't matter), clean it; if it might hide a real bug,
**flag it** rather than guessing at the right handling.

**4.3 Redundant guards on guaranteed paths.**
```
if arr and len(arr) > 0:          # `if arr:` already covers empty
if x is not None:                 # right after an early return that guarantees x
```
Remove guards the type system or control flow already guarantees. For each
branch, ask: can the runtime ever reach it? If never, it is dead defensiveness.

**4.4 Generic error messages.**
```
raise Exception("An error occurred while processing the request")
```
A useful error is specific and carries context (ids, inputs, state). Improve the
message or, if you cannot tell what context belongs there, flag it.

Note the tension: models also *omit* genuinely needed handling (timeouts, empty
input). De-slopping error handling means normalizing to *appropriate* handling —
remove the redundant and the silent, preserve or flag the meaningful. Do not
strip a guard just because guards are often slop.

---

## 5. Structure and boilerplate

**5.1 Duplicated logic instead of reuse.** A fresh `fetchProductWithRetry` when
`fetchWithRetry` already exists. Reuse the existing utility. This requires reading
the surrounding code — the scanner cannot see it.

**5.2 Monolith functions.** One function mixing UI, business logic, and I/O.
Split along the natural seams — but only if that is within the task's scope;
otherwise flag it rather than embarking on a large refactor.

**5.3 Dead code and redundant imports.** Unused variables, parameters, and
imports; a module imported at the top and re-imported inside functions. Remove.

**5.4 `else` after `return`.**
```
if cond:
    return a
else:              # the else is dead weight
    return b
```
Flatten to `if cond: return a` then `return b`.

**5.5 Dependency creep.** Adding `date-fns` when the project already uses `dayjs`.
Use what is already there; flag a genuinely needed new dependency rather than
adding it silently.

---

## 6. Cosmetics

Low individual severity, high signal value — these are the surface tells.

**6.1 Emoji** in code, comments, or log lines (`logger.info("Done 🚀")`). Remove
unless the project deliberately uses them. An emoji in a comment is a near-certain
AI tell.

**6.2 Unicode hazards.** Em-dashes (`—`), en-dashes (`–`), smart quotes
(`' ' " "`), non-breaking spaces, zero-width spaces, and the ellipsis character
(`…`) inside source or strings. Normalize to plain ASCII (`-`, `'`, `"`, regular
space, `...`) unless the string is genuinely user-facing copy where the character
is intended.

**6.3 Leftover debug output.** `print()`, `console.log`, `fmt.Println` left in as
debugging residue. Remove. (Caveat: real, intended logging stays — judge by
whether it looks like observability or like a forgotten probe.)

**6.4 Over-polished user-facing strings.** Uniformly "calm, professional" error
text that reads like marketing. Minor; align with the app's real voice.

---

## 7. Correctness and security — flag, do not silently clean

These change behavior. Surface them in the report; do not edit them away as if
they were style.

**7.1 Hallucinated or wrong-ecosystem packages/APIs.** Imports of libraries that
do not exist (`aws-helper-sdk`, `jwt-secure-validator`) or a real package from the
wrong ecosystem. Verify every unfamiliar import against the project manifest /
lockfile. Unverifiable imports are the slopsquatting supply-chain risk — flag,
never assume.

**7.2 Context-inconsistency bugs.** Use of an undefined or never-assigned symbol
(`cache=cache` where `cache` is never defined), use-before-assignment,
redeclarations. Locally plausible, globally broken. Flag.

**7.3 Happy-path-only logic.** Network calls without timeouts, N+1 queries in a
loop, retries without backoff/jitter (retry storms), missing input validation.
Models are trained on code that works, not code that survives production. Flag the
gap; suggest the missing guard.

**7.4 Hardcoded config and secrets.** Embedded URLs, ports, retry counts, feature
flags, and especially credentials (`password = "admin123"`,
`api_key = "test_key_replace_this"`). Flag; do not "tidy" a secret in place.

**7.5 Injection-prone queries and missing auth.** SQL built by string
concatenation or f-strings with user input; endpoints missing an auth/permission
check that peers have. Flag with a parameterized-query / add-the-check suggestion.

The rule: if you cannot verify correctness locally, you cannot clean it — you can
only report it.

---

## Carve-outs

Most patterns above are legitimate in some context. These are the cases where the
"slop" reading is wrong and you should leave the code alone (and usually say
nothing).

Note the asymmetry from `SKILL.md` principle 4: these carve-outs are about
**code**, where a wrong cut costs a bug. Only the last three apply to comments,
and they are narrow on purpose — the comment doctrine does not have a "when in
doubt, keep it" mode.

- **Tiny-scope generic names.** `i`, `j`, `x`, `err`, `ctx`, `acc`, `ok`, `_` in
  short scopes are idiomatic, not slop. A three-line loop body does not need
  `sessionIndex`.

- **Role suffixes and codebase vocabulary.** `OrderManager`, `PaymentClient`,
  `AuthHandler`, `ChatRepository`, `RealtimeProcessor` — a role word on a domain
  noun is deliberate architecture vocabulary, exactly right in a layered service
  (handlers -> managers -> repositories, one client per dependency). Never flag
  these. The scanner intentionally excludes `manager`, `client`, `handler`,
  `service`, `repository`, `controller` for this reason. Slop is only the *empty*
  qualifier (`DataManager`) or a *bare* generic local (`manager := ...` holding
  one specific manager → name it `chatManager`).

- **Defensive code in critical or boundary paths.** Input validation at a public
  API boundary, retries around a flaky external call, guards in payment or auth
  code — appropriate, not over-defensive. The slop version is redundant guards on
  paths the code already guarantees, not all guards.

- **Genuine abstractions.** An interface with two real implementations, a factory
  behind an actual plugin system, a helper with several callers — these earn
  their keep. Count the call sites and implementations before collapsing.

- **Intended cosmetics.** Emoji in a CLI's deliberately playful output, Unicode
  in user-facing copy where the character is correct (`—` in prose, `…` in a
  localized string). Judge the target, not the character in isolation.

- **Directives.** License/copyright/SPDX headers, `Code generated by ... DO NOT
  EDIT.` banners, build tags, pragmas, linter and type-checker switches,
  `Deprecated:` markers, Go example `// Output:` blocks. The tooling reads them;
  leave them byte-for-byte, however many lines they take.

- **Paths the profile allows.** One-line invariants under a path listed in the
  profile's `## Comments` section, where deleting the line lets a plausible edit
  break the code silently. One line, English, stateless; everything else on the
  path is cleaned as usual. A path is allowed only by the profile — never by the
  number of comments already in it.

- **Docs the human asked for.** A doc-comment survives when the human asked for
  it in this session, and then still at one line unless they asked for more. A
  linter rule, a language convention or an already-documented package is not
  asking.

A comment that encodes a real contract — `// off by one: the API index is
1-based` — is not a carve-out. It still goes; it is the case 1.10 exists for, so
move its content before deleting it.

On code: when in doubt, do not cut. A false positive that removes deliberate
logic is worse than a missed nit, because it erodes trust in the whole pass. On
comments the default inverts — cut, and let version control hold what was there.
The skill exists to make code read like a careful human wrote it, and a careful
human leaves no comments behind while never deleting code they don't
understand.
