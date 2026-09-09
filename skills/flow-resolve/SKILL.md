---
name: flow-resolve
description: >-
  Give every review finding an explicit verdict, apply and verify the fixes,
  and drive the review loop to convergence, recording the outcome in the task
  ledger.
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

## The four verdicts

**fix** — the failure scenario is real, this change introduced it, and the cost
of removing it is proportionate to the risk.

**accept** — real, ours, and consciously not fixed. This requires naming *what
makes it acceptable*: "the only caller validates this at the transport layer",
"the constructor makes that state unreachable", "this costs a branch on the hot
path to guard an input the API does not accept". A bare "accepted" is
indistinguishable from "forgotten" three weeks later, which is exactly when it
matters. A blocker cannot be accepted without the user explicitly agreeing —
that is their risk to take, not yours.

**defer** — real, but the change did not introduce it. This carries the same
burden of proof a rejection does: the file and line at the base ref where the
defect already sits. Without that line it is not a deferral, it is a fix you did
not feel like doing, wearing a better label.

One shape looks inherited and is not. A latent defect this change makes
reachable for the first time — the helper nothing called before, the branch no
input could select until now — belongs to the change that woke it, however old
the line is.

**reject** — the finding is factually wrong.

A rejection stands on exactly one of four grounds, and nothing else counts:

- it misreads the code — quote the line that says otherwise;
- the state it needs cannot occur — show the type, constant or invariant that
  forbids it;
- this change already guards it — cite the guard;
- it is style with no observable behaviour, which is `flow-cleanup`'s business
  and not a defect at all.

**"Unlikely" is not one of them.** Concurrency, a nil on a path only an error
handler reaches, a cold cache, an absent optional field, zero read as missing,
an off-by-one on a boundary the code does not exclude, a retry storm — all of
these are reachable, and reachable findings go to `fix` or `accept` with a
reason. Rejecting them as speculative is how a real defect leaves the ledger
labelled "false positive", and nothing downstream ever looks at it again.

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

## How much robustness this code owes

The close call — real, minor, and the fix costs a branch plus some structure —
turns on what the code *is*, and that is written down rather than felt: the
profile's `## Criticality` section and the paths it lists.

In **core** code, where a defect reaches callers who cannot see it and a
rollback does not undo what was already written, a serious finding is fixed, and
a minor one is fixed too when it touches the durability of state or a contract
other code reads. In **peripheral** code — tooling, admin surfaces, scripts — a
serious finding can be accepted when the fix would drag structure in behind it.
**Service** code sits where the defaults above already are.

The profile's `may be traded` line outranks this ladder wherever the two meet:
it was written by someone who knows what this system tolerates, and that beats a
general rule. Where the profile has no criticality section at all, read
`service` and say so in the verdict — an assumption doing this much work should
be visible to whoever reads the ledger next.

## Applying the fixes

Make the **smallest change that removes the failure scenario**. Do not refactor
around a finding: everything you touch has to be re-reviewed, and a large fix
diff reopens questions the round had already closed.

"Smallest" needs a testable edge, because it is generous to whoever holds the
keyboard. A repair that has to reach outside the function the finding names — a
changed signature, a new type, an edit in a file the finding never mentioned —
is not a fix. Stop there and re-verdict: either `accept`, with "its safe fix is
out of proportion" as the reason, or a task of its own. The same applies when a
fix turns out to be a feature — a missing transaction boundary, a queue that
needs redesigning. Record it, tell the user, keep this change reviewable.

**Derive the fix from the trigger, not from the finding's wording.** A reviewer
is reliable about the symptom and much less reliable about the cause, so a
repair written from the description tends to guard the wrong door: the failure
stays reachable and a legitimate input starts being rejected instead. Reproduce
the trigger, then fix what the reproduction shows.

### The contract every fix meets

**Red before, green after.** For a blocker or a serious finding, write the test
its trigger describes and watch it fail *before* touching the code. A fix
applied against a test that was never red proves nothing — it may be asserting
something the fix never touched. That test then stays as the finding's
regression.

**Everything else stays green.** The baseline is the last full verification
recorded in the ledger's `## Verification`. A test that was green there and is
red now is this fix's doing; "unrelated" is a claim needing the same evidence
`flow-test` demands of "pre-existing".

**A fix that breaks a green test is reverted, not repaired.** This is the rule
that keeps the loop finite. Patching the repair leaves the next round reviewing
code written twice under pressure, and that code reliably opens more findings
than it closed. Undo it, re-verdict the finding — `accept` with the regression
as the reason, or a task of its own — and record what happened. A blocker is the
one case that cannot end there: it goes to the user, unfixed and described.

**Blockers and serious findings land one at a time**, each verified before the
next; minor ones can batch by file. The asymmetry is about attribution — when a
batch of six turns something red, all you have learned is that the batch is bad.

When the fixes are in, run the full verification through `flow-test` and record
it. That run is the next round's baseline.

## Record

For each finding, append the verdict and its reason to the existing entry in the
ledger — `~/.claude/projects/<project-root-as-dashes>/flow/<task-slug>/ledger.md`,
schema in `../flow/references/ledger.md` relative to this skill's directory. Never rewrite or delete the finding text: the settled list is what the
next review round is handed, and it only works if it is complete — deferred
entries included. Leave one out and the next round rediscovers the same
inherited defect, which is the failure mode nobody spots, because the ledger
looks tidy and the loop simply never ends.

## Deciding whether to loop

Run another `flow-review` round when this one produced fixes worth re-examining.
**Stop when a full round across all lenses adds no new blocker or serious
finding that this change introduced.** Minor findings alone do not justify
another round — they justify a line in the report. Neither does an inherited
one, however severe: a defect the change did not cause will not converge by
being looked at again.

Before deciding, mark the round's arrivals: a finding whose anchor sits in code
a previous round's fix wrote has origin `from-fix`, and the reviewer cannot know
that — you can, from the ledger. It counts as `introduced` under every rule
above; the separate name exists so the loop can see where its own work is coming
from.

Three guards against spinning:

- **A loop that feeds itself stops.** Each round, count the findings closed
  against the new blocker and serious findings marked `from-fix`. When the
  second number matches or beats the first for two rounds running, the repairs
  have become this change's main source of defects and another round will not
  help. Stop, hand the user the open list, and say that plainly — it is the one
  thing they cannot see in the diff and the only basis on which they can decide
  to reset the fixes instead of continuing them.
- **A contested finding earns a witness before a third round of prose.** When a
  rejection comes back unchanged, stop arguing and write the test its trigger
  describes. Red, and the finding was real — flip the verdict to `fix` and keep
  the test as its regression. Green, and the rejection is evidence rather than
  opinion; delete the test unless it pins something worth pinning. This is the
  only move in the loop that produces a fact, and it costs less than the round
  it replaces.
- **A rejected finding that survives its witness and returns anyway** goes to
  the user, not into a fourth rebuttal. Either the reviewer sees something you
  do not, or the code needs to make its invariant explicit enough that a fresh
  reader stops tripping over it. Both outcomes are better than arguing with a
  subagent.

Nothing else runs between rounds. In particular, a round's fixes are **not**
cleaned before the next review: reviewers are told that comments, formatting
and naming are out of their scope, so unclean fixes cost the review nothing,
while a cleanup per round costs a fresh subagent and a full verification each
time for code the next round is about to rewrite. The change was cleaned once
before the first review; the only other pass is the narrow one at exit, below.

## When the run is unattended

If the user handed over the goal and left, three rules replace the conversations
you would otherwise have:

- **A blocker this change introduced cannot be accepted, only fixed** — and if
  it cannot be fixed inside this change, the run stops and reports rather than
  continuing. Lowering a severity so that the loop can finish is the one move
  that turns an honest unattended run into a dishonest one. An inherited blocker
  does not stop the run: it was there before anyone left, and halting a night's
  work over it would be theatre. It is deferred, and it leads the report.
- **Deferred findings are never fixed and never proposed.** Nobody is there to
  weigh a second task against the one they asked for, so the whole inherited
  list waits until morning.
- **The loop gets a budget** — the `budget:` in the ledger header, or three
  rounds when it does not set one. Convergence is the goal, but an unconverged
  run that stops and says what is still open is far more useful than one that
  spends the night rewording minor findings.

Everything else is unchanged: verdicts still get reasons, and the closing report
still leads with what was consciously accepted — with an unattended run it is
the first thing the user reads when they come back.

## Exit cleanup

Whichever way the loop ends — converged, budget exhausted, or stopped because
the repairs had become the source of the findings — the round fixes are the one
stretch of code nobody cleaned: they landed under review pressure, which is
exactly the state that produces stray comments, defensive scaffolding and
hurried tests. The implementation itself was cleaned before the first review
and then reviewed; it is not cleaned again.

So before the closing report, when any round's fixes touched non-test code, run
`flow-cleanup` once more, narrowed to those fixes: the files and lines the
`fixed` verdicts name, plus the regression tests they added — those were red
before their fix and green after, so they pass the rule of value by
construction and stay. A loop that closed on accepts, rejects and deferrals
alone leaves the diff exactly as it was cleaned; skip the pass and say so in
one line.

The pass is proportional. A handful of fix hunks is cleaned by you, in your own
context, with the two anti-slop skills applied literally; only a loop that
rewrote a substantial part of the change earns a dispatched cleaner. Either way
the party that edits runs the full verification exactly once at the end of the
pass and writes that line into the ledger — it is the closing baseline. Nothing
runs after it: not `flow-test`, not the build, not the scoped tests the
cleaner's report lists. The pass is safe after the final review precisely
because cleanup is bound to preserve behaviour and ends in that verification;
it does not reopen the review.

## Closing report

However the loop ends, give the user: how many rounds and why it stopped,
findings by severity,
and — the part they actually need — the two lists nobody can reconstruct from
the diff. **What was consciously accepted, with reasons**, which is the honest
description of the change's remaining risk and belongs to whoever wrote it. And
**what was deferred as already broken**, each with the line at base that proves
it.

The second list is a standing offer, not a plan. Interactively, ask once whether
any of it should become a task of its own and take silence as no. Never fold it
into this change: that is how a reviewable diff turns into a cleanup nobody
asked for, and the review round that already passed no longer covers it.
