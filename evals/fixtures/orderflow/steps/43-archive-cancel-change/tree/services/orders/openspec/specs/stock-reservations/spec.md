# stock-reservations

## Purpose

Everything the order flow asks of the inventory service. Orders never count
stock themselves; they hold it and give it back, always under an order id.

## Requirements

### Requirement: A reservation is keyed by order id

Every call to inventory SHALL carry the order id it is made for, so that
repeating a call cannot hold or release stock twice.

#### Scenario: The same reservation arrives twice

- **WHEN** the same order is reserved a second time
- **THEN** inventory holds the items once

### Requirement: An order the shelf cannot cover is refused

The service SHALL treat a refused reservation as a failure of the order, not as
something to fix up afterwards.

#### Scenario: One line is short

- **WHEN** one line of an order cannot be covered
- **THEN** no line of that order is held and the order is not stored

### Requirement: Stock follows the order out of the active states

Stock held for an order SHALL be released when the order stops being active, and
a release that failed SHALL be repeatable without further state changes.

#### Scenario: A cancelled order gives its stock back

- **WHEN** an order is cancelled
- **THEN** its reservation is released after the new status is stored

#### Scenario: Inventory is unreachable during a cancel

- **WHEN** the release fails
- **THEN** the order stays cancelled and the next cancel sends the release again
