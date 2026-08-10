# Add cancel order

## Why

Support pulls orders by hand a few times a week: a customer changes their mind
between placing an order and the pick list being printed. There is no endpoint
for it, so an operator edits the stored order, and the stock that order holds
stays reserved until somebody notices the shelf is short.

## What changes

- `POST /orders/{id}/cancel` moves a pending or paid order to `cancelled` and
  records why and when.
- Cancelling an order that is already cancelled answers 200 and changes nothing,
  so support can safely retry a request that timed out.
- An order that shipped or was delivered is refused with 409. That stock is
  physically gone; returns are a different flow with different paperwork.
- The reservation the order holds is released in inventory. The release is keyed
  by order id, so repeating it is harmless.

## Out of scope

- Refunds. Cancelling says nothing about money that already moved, and the
  payment provider has its own idea of when a charge can still be voided.
- Cancelling single lines of an order.

## Deltas

- `order-lifecycle`: one added requirement — which states an order may be
  cancelled in, and how a repeat behaves.
- `stock-reservations`: one added requirement, what happens to held stock when
  an order stops being active.
