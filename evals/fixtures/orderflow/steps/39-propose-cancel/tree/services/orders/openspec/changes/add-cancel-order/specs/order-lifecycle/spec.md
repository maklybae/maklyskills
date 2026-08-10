# order-lifecycle

## ADDED Requirements

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
