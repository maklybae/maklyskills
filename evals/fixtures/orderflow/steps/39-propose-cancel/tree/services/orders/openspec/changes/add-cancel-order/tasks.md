# Tasks

- [ ] Add `cancelled` to the lifecycle constants and run the generator
- [ ] `core.CancelOrder`: validation, the pending and paid rule, the repeat that
      changes nothing
- [ ] Table-driven tests for the four outcomes: cancelled, repeated, refused,
      unknown id
- [ ] `POST /orders/{id}/cancel` and the 409 that `ErrNotCancellable` maps to
- [ ] Release the reservation in inventory once the new status is stored
- [ ] Decide what happens when the release fails: the order stays cancelled and
      the next cancel repeats the release
- [ ] Fold both deltas into `openspec/specs` when the change is archived
