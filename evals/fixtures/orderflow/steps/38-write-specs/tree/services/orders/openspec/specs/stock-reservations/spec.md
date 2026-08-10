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
