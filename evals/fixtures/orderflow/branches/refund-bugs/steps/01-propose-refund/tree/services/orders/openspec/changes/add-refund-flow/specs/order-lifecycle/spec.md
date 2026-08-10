# order-lifecycle

## ADDED Requirements

### Requirement: An order that was paid for can be refunded

Orders in `paid`, `shipped` or `delivered` SHALL be refundable, in part or in
full. An order in `pending` SHALL NOT be: nothing has been charged for it, and
stopping it is a cancel. A `cancelled` order SHALL NOT be either; its money was
never taken or has already gone back.

The sum of the refunds of an order SHALL NOT exceed what the order was worth.

#### Scenario: A line comes back

- **WHEN** a refund names one line of a delivered order
- **THEN** the refund is worth that line's units at the price the order paid

#### Scenario: Nothing is named

- **WHEN** a refund names no lines
- **THEN** it is worth everything that has not been refunded yet

#### Scenario: More than the order was worth

- **WHEN** a refund would take the total given back past the order value
- **THEN** it is refused and nothing is recorded

#### Scenario: An order nobody paid for

- **WHEN** a refund arrives for a pending order
- **THEN** it is refused as a conflict

### Requirement: A refund request carries an idempotency key

Every refund SHALL carry a key chosen by the caller, and a second request with a
key the order has already seen SHALL answer with the refund that key issued.

#### Scenario: The same request twice

- **WHEN** support sends the same refund request twice
- **THEN** the second answer is the first refund and the order holds one refund
