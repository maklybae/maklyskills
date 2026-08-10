# orderflow

Order management for the storefront. The `orders` service owns the lifecycle of
an order: what is in it, what it costs, and where it sits between placed and
delivered.

Standard library only. There is nothing to keep in step with, and the whole
thing builds on a laptop that has Go and nothing else.

## Running locally

```
go run ./services/orders/cmd/orders
```
