---
name: anti-slop-code
description: >-
  Detect and clean "AI slop" from code — the verbose, over-commented,
  over-engineered, cosmetically-polished patterns that language models tend to
  emit. Enforces a strict comment policy: zero comments by default, a hard
  one-line cap, and no doc-comments on public/exported symbols unless explicitly
  authorized. Use this skill whenever the user asks to de-slop, clean up,
  tighten, or humanize a chunk of code; whenever they say a piece of code looks
  "AI-generated", "too verbose", "over-commented", "over-engineered", or has
  "too many comments / needless abstraction / defensive junk"; whenever they
  mention "slop", "AI slop", "deslop", or "unslop"; and proactively right after
  generating a substantial chunk of code that the user wants tightened before
  commit or review. Prefer this skill over an ad-hoc cleanup so the comment
  policy and the clean-vs-flag safety split are applied consistently.
---

# Anti-Slop Code

## What slop is (and why it hides)

AI slop is code that looks finished but carries no weight per line: comments
that restate the code, generic names that name nothing, abstractions with one
caller, defensive checks the runtime can never reach, hollow adjectives, and
cosmetic tells (emoji, em-dashes, "Here's the function that..."). The insight
that makes this worth a skill: **slop is defined by polish, not by messiness.**
Human code has tells — typos, uneven formatting — that make a reviewer slow down
and scrutinize. Slop reads clean, so reviewers relax and skim, and the
over-engineering and happy-path thinking underneath slip through. Cleaning slop
is restoring the code to what an experienced engineer *who already knows this
codebase* would have written: no more, no less.

## The four principles (the spine of every decision)

1. **Minimal delta.** Touch only the slop. Do not rewrite working logic, rename
   things wholesale, or "improve" architecture that wasn't asked about. Every
   edit should be defensible as "this line was slop and now it isn't." A large
   diff means you drifted from the task. This includes idiomatic rewrites: a
   working block that is merely un-idiomatic — a manual loop that could be a
   comprehension, an explicit group-by that could be `setdefault` — is not slop.
   Leave it. Slop is weight without information, not a style a purist would
   restyle. Comments are the one place where a large delta is expected and
   correct: deleting them is always in scope.

2. **Coherence beats abstract best-practice.** Slop is deviation from *this
   codebase's* idioms, not violation of some universal rulebook. Before editing,
   read the surrounding code and match its conventions — naming, error style,
   structure. Never import a "best practice" that fights the local environment.
   The one convention that does **not** get this deference is comment density:
   a file full of doc-headers does not license you to keep them. Local habit
   loses to the comment doctrine below; the only exception is the published-API
   permission spelled out there.

3. **Clean the stylistic; flag the behavioral.** Stylistic slop (comments,
   names, dead abstractions, cosmetics) you clean directly. Anything that could
   change what the program *does* — hallucinated packages/APIs, missing
   timeouts/validation, swallowed exceptions hiding bugs, hardcoded secrets, SQL
   built by string concatenation — you do **not** silently edit. You surface it
   in the report for a human to decide. Silently altering behavior under the
   banner of "cleanup" is how a de-slop pass introduces a real bug.

4. **Doubt is asymmetric.** The two halves of this pass have opposite failure
   costs, so they get opposite defaults:
   - **On comments, when in doubt, cut.** A wrongly deleted comment costs one
     `git show` to recover. A wrongly kept comment costs every future reader,
     forever, and rots into a lie the first time the code changes underneath it.
   - **On code, when in doubt, leave it and note it.** A wrongly deleted guard,
     helper, or branch costs a production bug. Generic names in tiny scopes,
     defensive code in critical paths, an abstraction with several callers —
     these are legitimate; confirm against context before touching them.

   The instinct to leave things alone when unsure governs code. It does not
   govern comments.

## Workflow

1. **Scope.** Establish what to clean: the pasted snippet, named files, or a diff
   (`git diff`, branch-vs-main). If the user is de-slopping recent work,
   prefer the diff — clean what changed, not the whole tree.

2. **Pre-scan.** Run the deterministic scanner to get cheap, high-recall
   candidates before reading:
   ```
   python3 scripts/scan_slop.py <files-or-dir>
   git diff --name-only | python3 scripts/scan_slop.py --stdin-list
   ```
   Treat every hit as a *candidate*, never a verdict. The scanner is blind to
   over-abstraction, wrong abstractions, and happy-path logic — those only come
   out by reading. So the scan is a floor, not a ceiling.

3. **Read and judge.** Read the code top-to-bottom asking one question per line:
   *why does this exist?* Cross-reference the full catalog in
   `references/patterns.md` for the good/bad pairs and the carve-outs that keep
   you from false-flagging. Classify each finding as clean-now or flag-for-human.

4. **Clean with minimal delta.** Apply the stylistic fixes. Preserve behavior
   exactly. Match the file's existing idioms. Re-read your diff and confirm every
   hunk removed slop rather than just moved it.

5. **Report.** Summarize what you cleaned and, separately and prominently, what
   you flagged. Deliver this inline as your response — the report is the message,
   not a separate file. Do not create a `report.md` (or similar) unless the user
   explicitly asks for one. See the report format below.

## The comment doctrine

This is the strictest part of the pass and the highest-yield. Comments are the
most common slop tell, and unlike code, a wrong call here is cheap to reverse —
which is exactly why the bar is set where it is.

### Default: zero

**Code is self-documenting. Names and types carry the meaning.** A comment is an
admission that the code failed to explain itself; the first fix to try is always
a better name, a named constant, or a type — not a comment. A comment earns its
place only when the information genuinely cannot live in the code.

### Hard cap: one line

**Two or more consecutive human-readable comment lines is a defect.** Not a
preference, not a smell — a thing to fix before you hand the code back. Condense
to one line, or move the detail to `docs/` and leave a one-line pointer.

The reasoning: a comment that outgrows one line has stopped being a pointer and
started being prose. Prose in source drifts out of sync with the code beside it
faster than anything else in the file, and no reviewer diffs it. If the
explanation truly needs a paragraph, the paragraph belongs somewhere a reader
will find it deliberately.

The cap holds **even when every line is individually a legitimate why.** A
five-line block of genuine rationale is still a five-line block: pick the load-
bearing sentence, keep that, drop the rest.

The only exemption is blocks that are multi-line **by machine mandate**, which
you leave byte-for-byte intact:
- license / copyright / SPDX headers
- `Code generated by ... DO NOT EDIT.` banners
- build tags and pragmas (`//go:build`, `# type: ignore`, `/* eslint-disable */`)

### Public symbols get no doc-comment

**A doc-header on an exported or public symbol — type, interface, func, method,
class, module — is banned by default.** `// NewPaymentManager creates a new
PaymentManager.` is the canonical example: it is the signature, retyped.

Being public is not itself a reason to document. The reason a symbol is public
is that other code calls it; what those callers need is a name that says what it
does, which the doc-header is quietly substituting for.

Exactly two things authorize a public doc-comment:
1. **The human asked for it in this session** — "keep the godoc", "document the
   public API", or similar.
2. **The module is a published library whose public surface is already
   consistently documented.** You are matching an established external contract,
   not starting one. A single stray doc-header elsewhere in the file is not this.

A linter or a language convention is explicitly **not** authorization. Go's
"exported symbols should have a comment", `pydocstyle`, ESLint's `require-jsdoc`
— when these fire on a self-explanatory name, the linter is wrong and the
one-line cap still applies to whatever you keep.

### Relocate the why; don't just drop it

The strictness above is safe only because deleting a doc-header does not have to
mean losing its information. Before you cut, ask where the why actually belongs:

- **Into a name or type.** A comment explaining that a number is in cents is
  answered by `amountCents int64`. Once the name says it, the comment is
  restatement — delete it outright.
- **Down to the line that enforces it.** A rationale attached to a public
  function usually belongs at the single guard or branch it explains, where a
  reader meets it in context and where it cannot silently stop being true.
- **Out to `docs/`.** For anything that genuinely needs a paragraph.

```go
// Pay charges the account for an order. It rejects non-positive amounts
// because the upstream gateway silently treats them as full refunds.
func (m *OrderManager) Pay(ctx context.Context, accountID string, amountCents int64) (string, error) {
	if amountCents <= 0 {
		return "", ErrInvalidAmount
	}
```
```go
func (m *OrderManager) Pay(ctx context.Context, accountID string, amountCents int64) (string, error) {
	// the gateway treats non-positive amounts as full refunds
	if amountCents <= 0 {
		return "", ErrInvalidAmount
	}
```
The first sentence was the signature retyped. The second was real, non-obvious,
external behavior — so it survives, as one line, at the guard that exists
because of it. Net: two lines of public doc-header become one line of load-
bearing context. That is the shape of a correct fix.

### Delete outright

A comment that does any of these carries nothing worth relocating:
- Restates the code (`counter++ // increment the counter`; `# create a user`
  above `user = User()`).
- Translates a name (`// createTexture` above `createTexture()`).
- Narrates the generation (`// Now we loop through...`, `// First, we...`,
  `// Here's the helper that...`, `// As we can see`).
- Documents the trivially obvious (a docstring on `add(a, b)` saying "adds two
  numbers and returns the result").
- Makes a hollow claim (`// blazing-fast, production-ready`) or cites an
  unverified metric (`// 50% faster`) with no benchmark or mechanism behind it.
- Is a section banner (`######## INITIALIZATION ########`).
- Exists only to satisfy a convention or a linter.

### Keep only these

- A non-obvious **why**: a business reason, a deliberate trade-off, the rationale
  for a workaround, stated in one line at the point it applies.
- An **invariant or precondition** a reader could violate without knowing it —
  and only if a name or type cannot carry it instead.
- A **contract with an external system** whose behavior is not visible here.
- A **functional marker the tooling reads** (`// Deprecated:`, `//go:embed`,
  build tags, `# type: ignore`).

Two hard rules on anything you keep or write:
- **English only.** Rewrite non-English comments into English.
- **Stateless.** A comment states what *is*, never how the code got here. Strip
  "previously", "now that", "changed from", "fixes #123", "regression test for",
  ticket keys, PR references. History lives in version control. A ticket
  reference survives only as a supplement to a real contract description, never
  as the whole explanation.

### The pass is net-negative on comments

A de-slop pass that ends with more comment lines than it started has failed,
without exception. Relocating a why is a *move*, and it shrinks: two lines up
top become one line inside. If you find yourself adding a comment to explain a
deletion, delete the comment instead — the diff already explains it.

Before reporting, count: comment lines in, comment lines out. The report states
both.

## The slop catalog (summary)

Full detail, good/bad pairs, and carve-outs live in `references/patterns.md`.
Read it before a serious pass; the summary below is the map.

- **Comments** — see the doctrine above. The largest and highest-value category.
- **Naming** — content-free names (`data`, `result`, `temp`, `obj`, `payload`),
  empty qualifiers (`DataManager`, `InfoHandler`), overlong legalese names,
  inconsistent naming for one concept in a scope. Role suffixes on a domain noun
  (`OrderManager`, `PaymentClient`, `AuthHandler`) are **not** slop — that is
  architecture vocabulary; keep it.
- **Over-abstraction** — single-use helpers, speculative layers "just in case",
  factories/managers/interfaces with one implementation, wrong/forced
  abstractions (one config-driven mega-component instead of two clear ones),
  wrapper functions and pointless intermediate variables (`x := f(); return x`).
- **Error handling** — the two mirror-image slops: blanket `try/except` on every
  operation, and silently swallowed exceptions (`except: pass`,
  `catch { /* ignore */ }`) that hide bugs behind a clean surface; plus redundant
  guards on already-guaranteed paths and generic "an error occurred" messages.
- **Structure** — duplicated logic instead of reusing an existing utility,
  monolith functions mixing concerns, dead code, redundant imports, `else` after
  `return`.
- **Cosmetics** — emoji in code/comments/logs, em-dashes and other Unicode
  hazards (smart quotes, non-breaking spaces), leftover debug prints, banner
  comments, over-polished user-facing strings.
- **Correctness / security (FLAG, do not silently clean)** — hallucinated
  packages/APIs (slopsquatting), happy-path logic missing timeouts/validation/
  backoff, hardcoded config and secrets, SQL via string concatenation, missing
  auth on endpoints that need it. These change behavior; they go in the report.

## Clean vs flag

| Clean directly (stylistic, behavior-preserving) | Flag for human (behavior-changing) |
|---|---|
| Any comment run over one line; public doc-headers | Hallucinated or wrong-ecosystem imports |
| Restating/narrating/hollow comments; trivial docstrings | Missing timeout / validation / retry backoff |
| Content-free / inconsistent names | Swallowed exception that hides a real failure |
| Single-use helpers, dead abstraction layers | Hardcoded secrets / credentials |
| Redundant guards on guaranteed paths | SQL built by string concatenation |
| `x := f(); return x`, `else` after `return` | Missing auth / permission checks |
| Emoji, em-dashes, Unicode hazards, debug prints | Anything whose correctness you can't verify locally |
| Dead code, redundant imports | |

Comments never appear on the right-hand column: deleting a comment cannot change
behavior, so there is nothing to flag and nothing to hesitate over. Everything
else on the left is still subject to principle 4 — when a *code* fix sits on the
line, prefer flagging over silently cutting.

## Report format

Keep the report tight. A one-line comment ledger, then two sections:

```
Comments: <N> lines → <M> lines (<M-N>)

## Cleaned
- <file>:<line> — <what was slop> → <what it is now> (one line each; group trivial ones)

## Flagged for review (behavior — not changed)
- <file>:<line> — <the concern>, why it matters, suggested direction
```

The ledger exists because it is the one number that cannot be fudged by
narration, and it must not go up. If nothing was slop, say so plainly rather
than inventing edits — a clean pass that changes nothing is a valid and honest
outcome. Do not manufacture churn to look busy; that is itself a form of slop.

## Guardrails against false positives

These apply to **code**, where a wrong cut costs a bug. They do not soften the
comment doctrine. Before you flag or cut, check against
`references/patterns.md#carve-outs`. The recurring legitimate cases:
- Generic names in genuinely tiny scopes (`i`, `x`, `err`, `ctx`, `acc`).
- Role-suffixed domain types and the codebase's established vocabulary.
- Defensive code and detailed logging in critical or externally-facing paths.
- An abstraction that really does have multiple call sites or implementations.
- Machine-mandated comment blocks: license/SPDX, generated-file banners, build
  tags. These are the one carve-out that survives on the comment side.

When you're not sure whether a piece of *logic* is deliberate, leave it and note
it. The skill's job is to make the code look like a careful human wrote it — and
a careful human deletes every comment they can, and none of the code they don't
understand.
