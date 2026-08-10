# orderflow

Order management for the storefront. `orders` owns the lifecycle of an order,
`inventory` owns stock levels and the reservations that back an order. They talk
over HTTP and share nothing but the two helper packages under `pkg/`.

Both binaries are standard library only. `orders` keeps its data in memory or in
a single JSON file, which is enough for the volumes we see today; `inventory`
boots from a stock snapshot and keeps its counters in memory.

## Layout

| path | what lives there |
| --- | --- |
| `services/orders` | order lifecycle, HTTP API on `:8080` |
| `services/inventory` | stock levels and reservations, HTTP API on `:8081` |
| `pkg/xerrors` | error wrapping used by both services |
| `pkg/httputil` | request id, logging and panic recovery middleware |
| `tools` | code generators |

Inside a service the layering is always `internal/api` → `internal/core` →
`internal/store`. `AGENTS.md` has the rules a change is reviewed against.

## Running locally

```
go run ./services/inventory/cmd/inventory -snapshot services/inventory/testdata/stock.json
go run ./services/orders/cmd/orders -inventory http://127.0.0.1:8081
```

Without `-inventory` the orders service skips stock reservations, which is handy
when you only care about the order endpoints.

```
curl -s localhost:8080/orders -d '{"customer_id":"cust-17","items":[{"sku":"desk-lamp","quantity":1,"unit_price":4900}]}'
curl -s localhost:8080/orders/ord_1a2b3c/cancel -d '{"reason":"customer changed their mind"}'
```

## Development

Everything CI gates on goes through the Makefile; `.github/workflows/ci.yml` is
the authority on what has to be green before a change lands.
