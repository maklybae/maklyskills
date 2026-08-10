# Matching notes

Read alongside `refund-bugs.json`. A reviewer finding matches a record when the
description names the same underlying defect **and** the location overlaps
`[start_line, end_line]` of the same file. These are the cases where location
alone decides nothing.

## Records that share a location

| records | same lines | what tells them apart |
| --- | --- | --- |
| B3, X2 | the ceiling in `RefundOrder` | B3: the ceiling is the whole order value instead of the value less prior refunds (over-refund). X2: the ceiling admits an amount of zero, so refund records with caller text can be appended for ever. A finding about paying too much is B3; a finding about records that move no money is X2. |
| B2, D2, X7 | `refundAmount` | B2: the amount is pro-rated over units instead of priced from the lines. D2 (decoy): "no quantity check here" — the validator owns it. X7: no refund records how many units of a sku already came back, so the same units are returned under a second key. X7 is anchored on the per-line guard inside the function. |
| B7, X6, D8 | `returnStock` | B7: the return is retried although inventory does not key returns. X6: the return shelves units that never left the warehouse. D8 (decoy): the refund path sends no Release, so the order's hold stays standing — that is the intended division of labour with the cancel path. A finding about the loop is B7; about restocking goods that never shipped, X6; about the missing release, D8. |
| D5, D6 | `refundByKey` | D5 (decoy): the scan runs backwards, which is not observable. D6 (decoy): the key is matched without a fingerprint of the request body. Both are false positives; they are recorded apart so a subject that reports one is not credited for the other. |
| C6, C7 | `JSONFile.write` | The same two statements, two consequences. C6: the order document is renamed into place before the refund book, so a crash between them leaves the book one refund behind and the rollback it exists for then loses that refund — this is the durability claim. C7: the document is already durable when the book fails, so the caller is handed a failed Create or Update for data that is on disk — this is the claim about the error the caller receives. A finding that names crash ordering is C6; one that names the caller acting on a false failure is C7. Both at once counts as both. |
| X9 | the `returnStock` call site in `RefundOrder` | The context handed to committed follow-up work. Distinct from B7 and X6, which are about the call itself. |
| C11, C15, D7 | the refund handler in `api/orders.go` | C11: a refund that is already on the order is answered as a failure, because the handler drops the `Refund` that `RefundOrder` returns beside the error. C15: the `order_id` in the answer is the raw path segment rather than the order the money came off. D7 (decoy): the refund lines the handler builds carry no price. A finding about the status code a committed refund gets is C11; about the id in the body, C15; about the missing price, D7. A finding pointing at `core/refund.go:109-111` and naming the failed answer is also C11. |
| C12, C23, X8 | the ship transition | C12: an order with a *partial* refund ships every unit it ever held. C23: the conditions that let an order through the guard are untested, the zero-value clause among them. X8 (refund-bugs): a *fully* refunded order ships, because that arm has no clause at all. A finding about goods leaving after a partial refund is C12; about a full refund on refund-bugs, X8; about the missing coverage, C23. Naming both the partial and the full case on refund-bugs counts as C12 and X8. |
| C13, C14 | `orderGate` and the call to `enter` | C13: the gate is held across `store.Update` and the HTTP call to inventory, and `enter` cannot see the caller's context. C14: the lock index is a signed conversion of a 32 bit digest and goes negative where `int` is 32 bits wide. A finding about queueing, throughput or a caller that has hung up is C13, wherever in `refund.go:41-49`, `64-67` or `109-111` it points; a finding about the index arithmetic is C14. |
| C10, C16 | `JSONFile.write`'s book half | C10: the book is rebuilt only from the orders the document still names, so restoring an older document drops the refunds of orders it predates. C16: the book is not written at all when no order has refunds, so taking the last refund away leaves the old book in place and the next read restores it. A finding about orders missing from the rebuilt book is C10; about a refund that comes back, or about the outcome depending on unrelated orders, C16. |
| C17, X5 | the return client's test | X5: the test proves only that the call reached a server that accepts anything - nothing pins the client it goes through or the request id it carries. C17: nothing pins the body either, so a payload that drops `quantity` ships green. A finding about the call is X5; about what the call carries, C17. |
| C21, B7, X6, D8 | `returnStock` on refund-bugs | C21's range sits inside the function B7, X6 and D8 share on that arm. C21 is a claim about the tests: no test reads what inventory was told for a refund that names no lines. A finding about the retry is B7, about restocking goods that never shipped X6, about the missing release D8, and about the missing assertion C21. |

## C2 and its three anchors

C2 is one defect with three places a reviewer can honestly point at, because the
window it describes has two ends and the gate that should close it is a third:

- `services/orders/internal/core/service.go:174-191` — `advance`'s
  load-modify-write, the recorded anchor;
- `services/orders/internal/core/refund.go:63-114` — the refund's own gate and
  write, the side the transition overwrites;
- `services/orders/internal/core/cancel.go:31-53` — `CancelOrder`'s
  load-modify-write, the same window by the other door.

A finding inside any of the three that names a transition overwriting a
committed refund is a hit on C2. A finding that only says "the gate is narrow"
without naming what is lost is not.

## Arm applicability

| records | refund-bugs | refund-clean |
| --- | --- | --- |
| B1-B9, X1-X14 | present | absent |
| C2, C11, C12, C15, C17-C21, C25 | present | present |
| C5, C6, C7, C10, C13, C14, C16, C22, C23, C24 | absent | present |
| D1-D9 | present | present |

The control arm is known, not clean: a finding against `refund-clean` that
matches a record whose `arms` names `refund-clean` is a hit, not a false
positive. A finding against either arm that matches a record whose `arms` does
not name that arm is a false positive. A finding that matches a decoy is a false
positive on both arms and is counted separately. A real pre-existing issue in
trunk code (anything outside the 27 files the branch touches) is a fixture
finding: route it to triage rather than counting it against precision.

Line numbers in a record are quoted from the arm the defect lives on: the C
records from `refund-clean`, everything else from `refund-bugs`. Records that
apply to both arms (C2, C11, C12, C15, C17-C21, C25, D1-D9) carry one arm's
numbers, and the same function sits at a different offset on the other arm, so
resolve those by the enclosing function named in the table above rather than by
the absolute line. The test-quality records are the sharpest case: C25 quotes
`refund_test.go:247-272` from `refund-clean`, where a different test occupies
those lines on `refund-bugs`, so on that arm resolve it by the claim - the
remainder of an order that has already been refunded in part is asserted
nowhere - and accept an anchor on `refundAmount`'s empty-lines branch.

## Findings that match nothing

These claims look like plants and are not. Each is settled in
`adjudications.md`; they are repeated here so a grader does not route them to a
neighbouring record.

- **A partial refund is netted against a later genuine return, so the return is
  refused**, reported beside C12. It is not: the netting only refuses units that
  have already been paid back, and a second refund of a sku the order held twice
  is priced and allowed. The half of that claim that holds is C12.
- **A fully refunded order is stuck in `paid` for ever**, reported against
  `advance`. A paid order is still cancellable, and a fully refunded one takes no
  further refunds, so both consequences the claim names are answered by the code.
- **The reason limit is counted in bytes rather than characters**, reported
  against `RefundRequest.validate`. The trunk's cancel flow measures its own
  reason with `len` and says "characters" in the same words; the refund flow
  copies its analogue. The neighbouring decoy about the allowance itself is D4.
- **`validate` puts no cap on the number of refund lines**, reported beside the
  key and reason caps. The body is capped at 1 MiB on the way in, a repeated sku
  is refused at the second line, and a sku the order never had is refused at the
  first, so no request buys more work than the body it paid for.
- **Every store call now parses a second file and the refund ledger is never
  pruned.** `JSONFile`'s own comment says it reads and rewrites the whole
  document on every call and that this is fine at this volume; the book is the
  same trade. Same shape as D3.
- **A whole-order refund is stored with an empty `lines` array**, so the record
  cannot say which units the money covered. A refund that names no lines is
  money only by design.

Two further claims match nothing for reasons of their own:

- **The gate map grows without bound**, reported on `refund-bugs` against
  `services/orders/internal/core/refund.go:29-52`. It does not match C8. That
  defect was clean-only — the clean arm took the gate before the order was
  loaded, so an invented order id was enough to add an entry — and it has since
  been fixed, so no C8 record exists. On `refund-bugs` the gate is entered after
  the store has answered, so only ids of real orders reach the map: the finding
  is a false positive there. The pin
  `witnesses/c8_an_unknown_order_costs_nothing_test.go` holds the property on
  both arms.
- **A refund releases nothing / restocks goods that never shipped**, reported on
  `refund-clean`. The release rule was C1 and C3 and is fixed on both arms: a
  refund shelves shipped and delivered lines only and never touches the hold.
  The pins are `witnesses/c1_a_refund_leaves_the_hold_alone_test.go` and
  `witnesses/c4_an_order_worth_nothing_still_ships_test.go`; on `refund-bugs`
  the restocking half is still open and is X6.

## Absence-shaped findings

X3, X4, X5, X8, X11, X12, X13 and X14 are partly or wholly about something that
is *not* there. Their anchors are the nearest code that should have carried it:

- **X3** the api table entry whose name claims the ceiling but exercises the
  per-line guard.
- **X4** `TestJSONFileSurvivesAReopen`, which round-trips with the same build and
  therefore pins no field name.
- **X5** `TestRestockSendsTheReturnedLines`, which asserts only what the test
  itself supplied.
- **X11** the reason case that pins the refusing side of the limit only.
- **X13** `TestRefundAnswersWithTheRefund`, which never reads the order back, so
  the order view's `total` and `refunded` are asserted nowhere.
- **X14** `TestRefundOrderKeepsTheRefundWhenTheReturnFails`, which never reads
  what inventory was told, so the number of returns is asserted nowhere.
- **X8** `MarkShipped`, where the guard against shipping a refunded order would
  sit; a finding may reasonably point at `advance` instead — accept either.
- **X12** `RefundRequest.validate`, which no longer refuses the same sku twice.

The clean arm's test-quality records are anchored the same way. Three of them
quote production code because that is where the mechanism they leave unguarded
lives, and a finding on the test that should have covered it counts too:

- **C17** `TestRestockSendsTheReturnedLines`, which reads back the line count and
  nothing inside a line.
- **C18** `TestReturnRejects`, which ends on `assertUntouched` and so reads
  `Reserved` for a call that moves `OnHand`.
- **C19** the `append` that records a refund; a finding on
  `refund_test.go:185-245`, the two tests that come closest to storing a second
  one, counts as well.
- **C20** the store write of a refund; a finding naming the unused
  `testsupport.Store.FailUpdate` counts as well.
- **C21** the early return for a refund that names no lines; a finding on either
  test that drives that shape without reading `stock.Returns()` counts as well.
- **C22** `TestRefundOrderIsSerialisedPerOrder`, whose assertions on the stored
  order survive a lost update.
- **C23** `TestRefundedOrderDoesNotShip`, which covers the refusing branch only;
  a finding on the guard in `advance` counts as well, provided it names the
  missing coverage rather than the shipping itself, which is C12.
- **C24** the refund book's own test, which writes one refund through `Create`.
- **C25** `TestRefundOrderRefusesARefundOfNothing`, which reaches the remainder
  only where it is zero.

A finding that names the missing test or guard and points anywhere inside the
anchor block is a hit. A finding that only says "tests are thin" without naming
what is unprotected is not.

X13 and X14 are test-quality records and B4 and B7 are the code defects they
leave uncovered. A subject that reports the code defect is credited for that one
only; a subject that says "and no test would notice" is credited for both. The
same pairing holds on the clean arm: C22 is the gap that leaves C13's gate
unpinned, and C23 the gap beside C12's shipping rule.

## X9 and B6

X9 (committed work runs on the request context) is observable through the core
flow: cancel the context and the return never leaves. At the client level B6
masks it, because `http.Post` ignores the context altogether. B6's witness
therefore asserts the bounded call and the request id header, not cancellation.
A subject that reports either one on its own is credited for that one; a subject
that reports "the return ignores the context" against the client is B6.
