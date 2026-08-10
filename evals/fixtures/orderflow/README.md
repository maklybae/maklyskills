# orderflow fixture

A generator for **orderflow**, a small Go order-management repository used as a
controlled target for eval runs. Everything an eval asserts about it — the
layering, the rule file, the traps, the shape of the history, the defects on the
feature branches — is ground truth by construction. The specs live in
`evals/specs/orderflow-fixture.md` and `evals/specs/orderflow-seeded-bugs.md`.

## Use

```sh
./generate.sh /tmp/orderflow-check                         # trunk only, main checked out
./verify.sh   /tmp/orderflow-check                         # 39 checks

./generate.sh /tmp/orderflow-bugs --branch refund-bugs     # trunk + the seeded feature branch
./verify-branch.sh /tmp/orderflow-bugs refund-bugs         # 26 checks, witnesses included

./generate.sh /tmp/orderflow-clean --branch refund-clean   # the control branch
./verify-branch.sh /tmp/orderflow-clean refund-clean

./witnesses/run.sh /tmp/orderflow-bugs                     # all witnesses against a checkout
```

A materialised repository is disposable: generate it under `/tmp`, throw it away
after the run, and never commit one into this bundle.

`generate.sh` builds the history commit by commit in a scratch directory and
then clones the target from it, so what an eval subject sees is an ordinary
checkout — packed objects, an `origin` remote, a reflog that starts with the
clone. With `--branch`, the branch commits are applied on `feature/refund` on
top of the trunk and that branch is what the clone checks out, with `main` left
in place as the merge base. Two runs produce the same SHAs: every author,
committer and date is fixed data under `steps/`, and nothing reads the clock or
a timezone database at generation time. Without the flag the result is
byte-identical to what it was before branches existed.

`verify.sh` and `verify-branch.sh` re-run `generate.sh` into temporary
directories as their last check, so they need the bundle to be intact, not just
the repository.

## Layout

```
generate.sh                     materialises trunk, optionally with a branch
verify.sh                       gate for a trunk-only materialisation
verify-branch.sh                gate for a branch materialisation
steps/                          the trunk history, one directory per commit
branches/refund-bugs/steps/     the refund feature with the seeded defects
branches/refund-clean/steps/    the same feature with the defects fixed
ground-truth/refund-bugs.json   the answer key: what is recorded, where, and why
ground-truth/matching-notes.md  what the grader needs to tell overlapping records apart
ground-truth/adjudications.md   every claim ever raised, its verdict, and the evidence
witnesses/                      one witness per defect, plus the pins and run.sh
```

`ground-truth/` and `witnesses/` never enter a materialised tree. The witness
runner copies a witness into a checkout, runs that one test, and removes it
again; `verify-branch.sh` fails if either ever shows up in the repository.

## steps/

One directory per commit, applied in name order. The branches use the same
format:

```
steps/NN-slug/
  step.env       AUTHOR_NAME, AUTHOR_EMAIL, AUTHOR_DATE, COMMIT_DATE
                 dates carry the author's real offset for that instant
  message.txt    the commit message, subject and body
  tree/          files this commit adds or replaces, at their repository paths
  delete.txt     optional, one repository path per line to remove
  revert-of      optional, the slug of the commit this one reverts; generate.sh
                 substitutes that commit's SHA into __REVERT_SHA__ in the message
```

A step's `tree/` holds whole files, not patches, so a file that changes three
times appears in three steps. Editing a step changes every SHA after it, which
is expected — the contract is that two runs of the same bundle agree, not that
the SHAs are stable across edits of the bundle.

The Go sources under `steps/*/tree` are gofmt-clean except for one file in
`18-stock-service`, which carries a hand-edit artefact that the next commit
fixes. That is deliberate; do not "fix" it in place.

## The two arms

Both branches add the same refund feature across the same five commits, by the
same two people on the same days, touching the same 27 files. The file list and
the commit subjects are identical and no file's length differs by more than its
declared budget, so the shape of a checkout does not say which arm it is.

The control arm is **known, not clean**. `refund-clean` is the arm without the
twenty-three seeded defects, not an arm without defects: what a review of it
turned up was adjudicated, and what survived adjudication is recorded rather
than repaired. Every record therefore carries an `arms` field:

| records | refund-bugs | refund-clean |
| --- | --- | --- |
| B1-B9, X1-X14 | present | absent |
| C2, C11, C12, C15, C17-C21, C25 | present | present |
| C5, C6, C7, C10, C13, C14, C16, C22, C23, C24 | absent | present |
| D1-D9 (decoys) | present | present |

Every bug record has a witness: a test, or for the records about test quality a
short mutation script, that is green when the property holds and red when the
defect is there. A witness declares the arms it is meant to be red on in a
`witness:red` header, and `verify-branch.sh` asserts that it stands exactly
there — red on the arms its record names, green everywhere else. A witness whose
mechanism does not exist on an arm (the refund book on `refund-bugs`) skips, and
a skip counts as green.

Four witnesses declare `witness:red none`. They are pins: defects that were
found, adjudicated real, and fixed on both arms, kept as tests so the fix cannot
quietly come undone. `ground-truth/adjudications.md` says which claim each one
came from.
