# Tasks

- [x] `POST /stock/returns` in inventory: take a set of lines back onto the shelf
- [x] `core.RefundOrder`: the paid, shipped and delivered rule, the amount of a
      partial return, the repeat that pays nothing
- [x] Record refunds on the order so the total given back can be read off it
- [x] Send the returned lines to inventory once the refund is stored
- [x] `POST /orders/{id}/refund` and the status codes the new domain errors map to
- [ ] Feed the finance export from the recorded refunds
- [ ] Archive this change once the export lands
