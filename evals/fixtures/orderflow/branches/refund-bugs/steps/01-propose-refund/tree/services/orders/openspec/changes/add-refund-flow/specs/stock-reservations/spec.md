# stock-reservations

## ADDED Requirements

### Requirement: Refunded lines go back on the shelf exactly once

The lines of a refund SHALL be sent to inventory after the refund is stored, and
SHALL be sent once. Inventory counts a return rather than keying it, so a resend
raises the shelf a second time and a repeated refund request SHALL NOT send its
return again.

#### Scenario: A refunded line is returned

- **WHEN** a refund names a line
- **THEN** inventory is asked once to add those units back

#### Scenario: A refund names no lines

- **WHEN** a refund names no lines
- **THEN** no return is sent: the money goes back and the shelf is left alone

#### Scenario: The return fails

- **WHEN** inventory refuses or does not answer
- **THEN** the refund stays recorded and the return is not retried by the
  service; the warehouse count settles it
