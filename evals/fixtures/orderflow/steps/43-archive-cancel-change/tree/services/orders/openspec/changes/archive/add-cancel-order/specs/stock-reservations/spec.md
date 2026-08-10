# stock-reservations

## ADDED Requirements

### Requirement: Stock follows the order out of the active states

Stock held for an order SHALL be released when the order stops being active, and
a release that failed SHALL be repeatable without further state changes.

#### Scenario: A cancelled order gives its stock back

- **WHEN** an order is cancelled
- **THEN** its reservation is released after the new status is stored

#### Scenario: Inventory is unreachable during a cancel

- **WHEN** the release fails
- **THEN** the order stays cancelled and the next cancel sends the release again
