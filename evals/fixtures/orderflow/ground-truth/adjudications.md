# Adjudications

Every claim ever raised against either arm, the verdict, and the evidence that
decided it. A claim that appears here is settled: it is not re-litigated, and a
later reviewer that raises it again is scored against the verdict recorded here.

Verdicts:

- **real** — a defect. Either it was fixed, or it became a record in
  `refund-bugs.json` with an `arms` field naming the arms it lives on.
- **INTENDED** — the code does what the change's spec delta asks for. Reporting
  it is a false positive; where it is likely enough to be raised again it is
  also recorded as a decoy.
- **WRONG** — the claim does not hold on the arm it was made against.
- **fixture defect** — the eval material was at fault, not the branch.

The fix budget is spent: both arms have taken their two hardening rounds. From
here every adjudicated finding lands as ground truth and nothing in the branch
steps is edited.

## Batch 1 — extras found on refund-bugs (run 1)

| # | claim | verdict | evidence |
| --- | --- | --- | --- |
| 1.1 | a refund that names no lines never restocks anything | INTENDED | the stock-reservations delta sends a return only when a refund names a line; the money-only comment in `returnStock` states it in the code, which retired this whole class of false positive |
| 1.2 | a same-key retry does not re-attempt a return that failed | INTENDED | the delta's no-resend clause: a key the order has already seen answers with the refund that key issued and sends nothing |
| 1.3 | the key is checked before the lock that protects the write | real → **X1** | two callers under one key both read an order with no refund, both compute one, and both write in turn |
| 1.4 | an amount of zero passes the ceiling, and the key has no length bound | real → **X2** | a fully refunded order takes an unbounded number of records carrying caller text |
| 1.5 | the api case named for the ceiling exercises the per-line guard | real → **X3** | it asks for nine of a line the order holds one of, so `ErrRefundTooLarge` is asserted nowhere |
| 1.6 | nothing pins the field names of a stored order document | real → **X4** | the file backend is only tested against documents the same build wrote |
| 1.7 | the return client test asserts what the test itself sent | real → **X5** | the server accepts anything and the assertions read back the request body |
| 1.8 | a paid order is restocked although nothing left the warehouse | real → **X6** | `returnStock` restocks for every refundable status; the reservation half of this claim is D8 |

## Batch 2 — extras found on refund-clean (run 2)

| # | claim | verdict | evidence |
| --- | --- | --- | --- |
| 2.1 | returned quantities are not netted against earlier refunds | real → **X7** | the same three trays pass twice under two keys and six units land on the shelf |
| 2.2 | an order refunded in full still ships | real → **X8** | `advance` gates on status alone and no transition asks what has been refunded |
| 2.3 | committed follow-up work runs on the caller's context | real → **X9** | cancel the context between the write and the return and the return never leaves; masked on refund-bugs by B6, because `http.Post` ignores the context altogether — B6's witness was re-cut to assert the bounded call and the request id header instead |
| 2.4 | a rollback erases the refund ledger | real → **X10** | refunds live only inside the order document, which an older build rewrites without them |
| 2.5 | the reason limit the change argued for is not pinned | real → **X11** | the only case sends 501 characters, which is refused at 500 and at 240 alike |
| 2.6 | `validate`'s guards are untested | real → **X12** | later converted to a code defect on refund-bugs by removing the duplicate-sku guard, so the record is the missing guard and the missing test together |

## Batch 3 — the eighteen claims

| # | claim | verdict | evidence |
| --- | --- | --- | --- |
| 3.1 | a refund gives the order's warehouse hold back | real → **C1**, fixed | the hold belongs to the cancel path; `returnStock` no longer calls `Release` on either arm. Pin: `witnesses/c1_a_refund_leaves_the_hold_alone_test.go` |
| 3.2 | a transition that overlaps a refund writes the refund away | real → **C2**, recorded, both arms | a deliver that read the order before the refund committed writes back a copy with no refunds; the key can then pay a second time. Claim 3.14 is the same defect reached through `CancelOrder` |
| 3.3 | the release is all-or-nothing by order id, so one refunded line gives the whole hold away | real → **C3**, fixed | same edit as 3.1: with no release in the refund path the question does not arise. Covered by the C1 pin |
| 3.4 | an order worth nothing can never ship | real → **C4**, fixed | the guard against shipping a refunded order read `refunded >= value`, which holds at zero for an order that has been refunded nothing; it now requires `value > 0`. Pin: `witnesses/c4_an_order_worth_nothing_still_ships_test.go` |
| 3.5 | a return that failed is never offered again | INTENDED | the delta's no-resend clause; the claim also conflated `Release` with `Restock`, which are different calls with different owners |
| 3.6 | a refund book that cannot be read is read as an empty one | real → **C5**, recorded, refund-clean | the length test runs before the error test, so every failed open returns `(nil, nil)` |
| 3.7 | the refund book is written after the document it backs up | real → **C6**, recorded, refund-clean | a crash between the two renames leaves the book one refund behind, and the rollback it exists for then loses that refund |
| 3.8 | a store call fails after it has already committed | real → **C7**, recorded, refund-clean | the document is durable before the book is attempted, so `PlaceOrder` releases stock and fails the client for an order that is on disk |
| 3.9 | the gate map grows for every order id the caller invents | real → **C8**, fixed, clean-only | the clean arm took the gate before loading the order, so an id that does not exist was enough to add an entry; a fixed 64-lock stripe replaced the map. Pin: `witnesses/c8_an_unknown_order_costs_nothing_test.go` |
| 3.10 | a refund whose lines price to zero is refused as having nothing left | real → **C9**, fixed | the zero-amount guard now also requires that the request named no lines. Pin: `witnesses/c9_a_line_that_cost_nothing_is_refundable_test.go` |
| 3.11 | an idempotency key is matched without the request that carried it | INTENDED → decoy **D6** | the delta asks for exactly this; a body fingerprint and a conflict on mismatch is a design change |
| 3.12 | the refund book is rebuilt only from the orders the document still names | real → **C10**, recorded, refund-clean | restoring an older `orders.json` drops the refunds of every order that predates it |
| 3.13 | nothing reads an order back over http after a refund | real → **X13** | the order view's `total` and `refunded` are asserted nowhere on refund-bugs; refund-clean now carries the assertion |
| 3.14 | `CancelOrder` overwrites a refund that committed while it was in flight | real → **C2** | the same window as 3.2 by the other door; recorded once, with `cancel.go:31-53` listed as an anchor in the matching notes |
| 3.15 | the store stub's `HoldUpdates` hook is dead weight that changes `Update`'s timing | fixture defect | the hook existed only for one witness; it was removed from both arms and the witness rewritten around a store of its own |
| 3.16 | the gate map grows without bound (raised against refund-bugs) | WRONG | refund-bugs enters the gate after the store has answered, so only ids of real orders reach the map; on that arm the finding is a false positive |
| 3.17 | nothing counts how often the return is sent | real → **X14** | the failing-return test reads no inventory state, so `restockAttempts` can be raised to 100 with the suite green |
| 3.18 | a stored refund line carries a unit price of zero | WRONG as a defect → decoy **D7** | nothing reads the price on a refund line: the amount is priced from the order's items and the return sends skus and quantities |

## Batch 4 — the three review runs on refund-clean (runs 9, 10 and 11)

Three reviews of `refund-clean`, deduplicated across the runs. The recurrence
column counts how many of the three raised the claim; the graded hits (C2, C5,
C6, C7 and the D8 decoy) are not repeated here.

| # | claim | seen | verdict | evidence |
| --- | --- | --- | --- | --- |
| 4.1 | a refund that is already on the order is answered as a failure, and the handler drops the `Refund` returned beside the error | 3/3 | real → **C11**, both arms | probed over http: the store holds 4900 and the answer is `500 internal`; the repeat under the same key answers 200 with that refund. The no-resend clause of the stock delta covers the return that is not retried, not the answer the caller is handed for money that moved |
| 4.2 | a partly refunded order ships every unit it ever held | 3/3 | real → **C12**, both arms | refund one of two lamps on a paid order and `MarkShipped` succeeds with `Items` untouched; the guard fires only at `Refunded() >= value`. The second half of the claim, that the netting then refuses the genuine return, is **WRONG**: the second lamp is priced and refunded without complaint |
| 4.3 | the order gate is held across the inventory call and `enter` is blind to the context | 3/3 | real → **C13**, refund-clean | a refund stalled inside `Restock` holds its stripe: a second caller whose context is already cancelled waits behind it indefinitely. On refund-bugs the gate is released before the return goes out, so the same witness is green there |
| 4.4 | `int(digest.Sum32()) % len(g.locks)` is negative on a 32 bit build | 2/3 | real → **C14**, refund-clean | modelling a 32 bit `int` with `int32` and entering the gate for an id whose digest has the high bit set panics out of range; about half of all ids qualify |
| 4.5 | the reason limit is enforced in bytes, not characters | 1/3 | INTENDED | `CancelRequest.validate` in the trunk measures its reason with `len` and words the message "characters" identically; the refund flow copies its analogue, and the byte count is the repository's convention for text limits. Not recorded as a decoy: one run in three, and D4 already sits on the allowance itself |
| 4.6 | the only wire test of the return asserts the line count and nothing inside a line | 3/3 | real → **C17**, both arms | dropping `quantity` from the return payload leaves every package green on both arms; X5 is the neighbouring record about the call rather than its body |
| 4.7 | `assertUntouched` inspects only `Reserved`, which a return never moves | 2/3 | real → **C18**, both arms | booking the lines before validating them keeps all three cases of `TestReturnRejects` green on both arms |
| 4.8 | no test stores two refunds on one order | 1/3 | real → **C19**, both arms | replacing the append with an assignment that keeps the newest refund only is green on both arms |
| 4.9 | `FailUpdate` is never used, so the failed write of a refund is undriven | 2/3 | real → **C20**, both arms | swallowing the error from `store.Update` is green on both arms; cancel has `TestCancelOrderKeepsStockWhenTheStoreFails` and refund has no counterpart |
| 4.10 | the stock outcome of a refund that names no lines is unpinned | 1/3 | real → **C21**, both arms | deleting the money-only early return is green on both arms |
| 4.11 | the serialisation test catches the loss of the gate only by timing | 1/3 | real → **C22**, refund-clean | with `enter` reduced to a no-op the test passed 20 of 30 runs on this machine, and 15 of 15 under `GOMAXPROCS=1`; the assertions on the stored order cannot fail at all, because a lost update leaves exactly one refund |
| 4.12 | only the refusing branch of the ship guard is tested | 2/3 | real → **C23**, refund-clean | dropping the `value > 0` clause is green: no test builds an order worth nothing. The other half of the claim, that nothing asserts a partly refunded order still ships, is answered by C12 - that it ships is the defect |
| 4.13 | the refund book is tested for one refund written through `Create` | 1/3 | real → **C24**, refund-clean | truncating the book to the first refund of each order is green in both store modes |
| 4.14 | the remainder after a partial refund is asserted nowhere | 1/3 | real → **C25**, both arms | returning nothing for an unnamed refund once anything has been refunded is green on both arms; the two tests that drive that shape sit at the ends of the range |
| 4.15 | the refund book is never truncated, so a removed refund comes back | 1/3 | real → **C16**, refund-clean | an order overwritten without its refund keeps that refund when it is the only refunded order and loses it when any other order has one, so the outcome turns on unrelated data |
| 4.16 | every store call parses a second file and the ledger is never pruned | 1/3 | INTENDED | `JSONFile`'s own comment says every call reads and rewrites the whole document and that this is fine at this volume; the book is the same trade at the same volume. Same shape as D3, no new record |
| 4.17 | a fully refunded order is stuck in `paid` when a ship notification retries | 1/3 | WRONG | a paid order is still cancellable, so there is a path out, and a fully refunded order takes no further refunds, so the second consequence the claim names cannot arise |
| 4.18 | `validate` puts no cap on the number of refund lines | 1/3 | INTENDED | the body is capped at 1 MiB by `httputil.Decode`, a repeated sku is refused at the second line and a sku the order never had at the first, so the work a request can buy is bounded by the body it sent |
| 4.19 | a refund records no actor and the endpoint authenticates nobody | 2/3 | INTENDED → decoy **D9** | nothing in either service reads a caller, no stored type holds one, and the spec delta asks for a key, a reason and lines; the correct resolve verdict is accept, not reject |
| 4.20 | the raw path segment is echoed as `order_id` | 1/3 | real → **C15**, both arms | `POST /orders/ord_delivered%20/refund` answers `order_id` with a trailing space beside a refund id that names the real order |

## Leftovers

| claim | verdict | evidence |
| --- | --- | --- |
| a whole-order refund is stored with an empty `lines` array, so the record cannot say which units the money covered (refund-bugs) | INTENDED | a refund that names no lines is money only, by design: it is worth what is left of the order value and nothing comes back to shelve, which is why `returnStock` returns early on it. The units are the order's own items and the amount is the remainder; the record is not the place that names them. No record — the claim is not likely enough in this wording to need a decoy. Re-affirmed after batch 4; it is listed in `matching-notes.md` under findings that match nothing, and the neighbouring test-quality claim about that shape is C21 |
| the gate map grows without bound (refund-bugs) | WRONG | see 3.16. Called out in `matching-notes.md` so a grader does not route it to C8, which is clean-only and fixed |

## Retired ids

C1, C3, C4, C8 and C9 were real and are fixed on both arms. They carry no record
in `refund-bugs.json`; their witnesses stay in `witnesses/` as pins, green on
both arms, declaring `witness:red none`. The ids are not reused.

## Census after batch 4

52 records: 43 bugs and 9 decoys. Of the bugs, 23 live on `refund-bugs` only, 10
on `refund-clean` only and 10 on both. By lens: 16 tests, 9 correctness, 9
integration, 6 robustness, 3 security. By severity: 3 blocker, 16 serious, 24
minor. 47 witness files: one per bug record and four pins.
