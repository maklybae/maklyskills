# orderflow

Order management for the storefront. `orders` owns the lifecycle of an order,
`inventory` owns stock levels and the reservations that back an order. They talk
over HTTP and share nothing but the two helper packages under `pkg/`.

Standard library only. There is nothing to keep in step with, and the whole
thing builds on a laptop that has Go and nothing else.

## Layout

| path | what lives there |
| --- | --- |
| `services/orders` | order lifecycle, HTTP API on `:8080` |
| `services/inventory` | stock levels and reservations, HTTP API on `:8081` |
| `pkg/xerrors` | error wrapping used by both services |
| `pkg/httputil` | request id, logging and panic recovery middleware |

Inside a service the layering is always `internal/api` → `internal/core` →
`internal/store`.

## Running locally

```
go run ./services/inventory/cmd/inventory -snapshot services/inventory/testdata/stock.json
go run ./services/orders/cmd/orders -inventory http://127.0.0.1:8081
```

Without `-inventory` the orders service skips stock reservations, which is handy
when you only care about the order endpoints.
