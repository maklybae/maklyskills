# order-lifecycle

## Purpose

What an order is and how it moves between the states we admit to: pending, paid,
shipped, delivered, cancelled. Every rule here is enforced in `internal/core`,
never in a handler.

## Requirements

### Requirement: An order needs a customer and at least one line

The service SHALL refuse an order with no customer, with no lines, with a
quantity that is not positive, or with the same sku on two lines.

#### Scenario: A valid order is stored

- **WHEN** an order arrives with a customer and one line
- **THEN** it is stored with status `pending`, an id and a placement timestamp

#### Scenario: The same sku twice

- **WHEN** two lines of one order name the same sku
- **THEN** the request is refused as an invalid request and nothing is stored

### Requirement: Placing an order holds the stock it needs

The service SHALL reserve the items of an order before the order is stored, and
SHALL release that reservation if storing the order fails.

#### Scenario: The shelf cannot cover the order

- **WHEN** inventory refuses the reservation
- **THEN** the order is not stored and the caller is told the stock is short

### Requirement: An order moves forward one state at a time

Only `pending` orders become `paid`, only `paid` orders become `shipped`, only
`shipped` orders become `delivered`. A transition that already happened SHALL be
treated as done rather than as an error.

#### Scenario: An unpaid order cannot ship

- **WHEN** the warehouse marks a pending order as shipped
- **THEN** the request is refused and the order stays pending

#### Scenario: The same transition twice

- **WHEN** an order that is already shipped is marked shipped again
- **THEN** the stored order is returned unchanged

### Requirement: An order that has not left the warehouse can be cancelled

Orders in `pending` or `paid` SHALL be cancellable with an optional reason of at
most 240 characters. Orders in `shipped` or `delivered` SHALL NOT be, because
that stock has physically gone; returns are a different flow.

#### Scenario: A pending order is cancelled

- **WHEN** a cancel arrives for a pending order
- **THEN** the order becomes `cancelled` and keeps why and when it happened

#### Scenario: Cancelling twice

- **WHEN** a cancel arrives for an order that is already cancelled
- **THEN** the stored order is returned unchanged and the call succeeds

#### Scenario: A shipped order is refused

- **WHEN** a cancel arrives for a shipped order
- **THEN** the request is refused as a conflict and the order stays shipped
