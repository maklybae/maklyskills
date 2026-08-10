# Tasks

- [x] Add `cancelled` to the lifecycle constants and run the generator
- [x] `core.CancelOrder`: validation, the pending and paid rule, the repeat that
      changes nothing
- [x] Table-driven tests for the four outcomes: cancelled, repeated, refused,
      unknown id
- [x] `POST /orders/{id}/cancel` and the 409 that `ErrNotCancellable` maps to
- [x] Release the reservation in inventory once the new status is stored
- [x] Decide what happens when the release fails: the order stays cancelled and
      the next cancel repeats the release
- [x] Fold both deltas into `openspec/specs` when the change is archived
