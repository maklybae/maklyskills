# Add refund flow

## Why

Cancelling an order stops it before it leaves the warehouse. Anything after that
— a lamp that arrived scratched, a customer who sent one of three trays back —
is money that has to go back by hand today: support asks finance over chat,
finance moves the money, and nobody writes down which order it belonged to.

## What changes

- `POST /orders/{id}/refund` refunds part or all of an order that was paid for.
  The body carries an idempotency key, an optional reason and the lines that
  came back; no lines means everything that is left.
- A refund is recorded on the order, so what has already been given back is
  visible to anyone who reads the order.
- Repeating a request with the same key answers with the refund the first call
  issued rather than paying a second time.
- What comes back goes back on the shelf: inventory gains
  `POST /stock/returns`, which takes a set of lines and raises the counters.

## Decisions worth writing down

- **A refund reason may be 500 characters**, more than the 240 the cancel flow
  allows. Support pastes the part of the ticket that explains the refund, and
  finance reads it during the monthly reconciliation; truncating it to a cancel
  reason's length loses the half that matters.
- **A refund that names no lines gives money back only.** Nothing is said to
  have come back, so nothing goes on the shelf; a return the warehouse has not
  seen would be stock we invented.
- **Returns are counted, not keyed.** Inventory has no way to tell a resent
  return from a second parcel, so the order service sends each return exactly
  once and lets the warehouse count settle anything that goes missing.

## Out of scope

- Talking to the payment provider. This records the refund and moves the stock;
  the money still leaves through the finance export.
- Refunding an order that was never paid for. That is a cancel.

## Deltas

- `order-lifecycle`: one added requirement, when an order may be refunded and
  how much of it.
- `stock-reservations`: one added requirement, what happens to the lines a
  refund sends back.
