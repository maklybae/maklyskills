---
name: flow-resolve
description: >-
  Work through review findings one by one, giving each an explicit verdict — fix
  it, consciously accept it with a stated reason, or reject it as factually
  wrong with the code that disproves it — then apply the fixes, verify, record
  the verdicts in the ledger, and decide whether another review round is needed.
  Use this skill when the user says "исправь замечания", "разбери находки",
  "fix the review comments", "что делаем с этим ревью", when flow-review has
  just produced findings, or when review feedback has arrived from a human
  reviewer or a pull request. Prefer it over fixing findings straight down the
  list: mechanically implementing every suggestion is how a clean change
  accumulates defensive clutter nobody asked for.
---

# Flow resolve

Every finding gets a verdict. Silence is not a verdict — a finding left
unanswered comes back next round with the same words, and the loop stops
converging.

## The three verdicts

**fix** — the failure scenario is real and the cost of removing it is
proportionate to the risk.

**accept** — real, and consciously not fixed. This requires naming *what makes
it acceptable*: "the only caller validates this at the transport layer", "the
constructor makes that state unreachable", "this costs a branch on the hot path
to guard an input the API does not accept". A bare "accepted" is
indistinguishable from "forgotten" three weeks later, which is exactly when it
matters. A blocker cannot be accepted without the user explicitly agreeing —
that is their risk to take, not yours.

**reject** — the finding is factually wrong. Cite the code that disproves it.
"I disagree" is not a rejection; a line number is.

## The judgement, which is the actual work here

Reviewers systematically over-produce two kinds of suggestion, and both are
usually correct to accept rather than fix:

- **Guards against states something already prevents** — the type system, the
  constructor, the single caller, a validation one layer up. The guard costs a
  branch and a reader's attention forever, against a risk of zero.
- **Generality for a second case that does not exist** — a parameter, an
  interface, a hook, for the variant somebody might want later.

When you accept one of these, say which it is. That note is what lets a future
reader tell that it was weighed rather than skipped — and it is what stops the
next round from raising it again.

The mirror-image mistake is accepting a genuine defect because it is "an edge
case". The test is **reachability, not likelihood**. If user input, a retry, a
restart, or a concurrent request can put the system in that state, it is not an
edge case — it is an untested path, and rarity only means it will surface at the
worst possible time.

When a call is genuinely close, decide by what being wrong costs. A wrongly
accepted correctness finding costs an incident; a wrongly applied nitpick costs
three lines of clutter. That asymmetry is why blockers and serious findings
default to *fix* while minor ones default to *accept* unless the fix is nearly
free.

## Applying the fixes

Make the **smallest change that removes the failure scenario**. Do not refactor
around a finding: everything you touch has to be re-reviewed, and a large fix
diff reopens questions the round had already closed.

Work in batches by file, running scoped tests after each. When all the fixes are
in, run the full verification through `flow-test` and record it.

If a fix turns out to be a feature in its own right — a missing transaction
boundary, a queue that needs redesigning — it is a new task, not part of this
round. Record it in the ledger, tell the user, and keep the current change
reviewable.

## Record

For each finding, append the verdict and its reason to the existing entry in the
ledger — `~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`,
schema in `../flow/references/ledger.md` relative to this skill's directory. Never rewrite or delete the finding text: the settled list is what the
next review round is handed, and it only works if it is complete.

## Deciding whether to loop

Run another `flow-review` round when this one produced fixes worth re-examining.
**Stop when a full round across all lenses adds no new blocker or serious
finding.** Minor findings alone do not justify another round — they justify a
line in the report.

Two guards against spinning:

- **A rejected finding that returns unchanged twice** goes to the user, not into
  a third rebuttal. Either the reviewer sees something you do not, or the code
  needs to make its invariant explicit enough that a fresh reader stops tripping
  over it. Both outcomes are better than arguing with a subagent.
- **Substantial fixes earn a cleanup pass first.** Fixes arrive with fresh
  comments, defensive scaffolding and hurried tests; reviewing that is reviewing
  noise.

## When the run is unattended

If the user handed over the goal and left, two rules replace the conversations
you would otherwise have:

- **A blocker cannot be accepted, only fixed** — and if it cannot be fixed
  inside this change, the run stops and reports rather than continuing. Lowering
  a severity so that the loop can finish is the one move that turns an honest
  unattended run into a dishonest one.
- **The loop gets a budget**, three rounds by default. Convergence is the goal,
  but an unconverged run that stops and says what is still open is far more
  useful than one that spends the night rewording minor findings.

Everything else is unchanged: verdicts still get reasons, and the closing report
still leads with what was consciously accepted — with an unattended run it is
the first thing the user reads when they come back.

## Closing report

When the loop converges, give the user: how many rounds, findings by severity,
and — the part they actually need — **the list of what was consciously
accepted, with reasons**. That list is the honest description of the change's
remaining risk, and it is the one thing nobody can reconstruct from the diff.
