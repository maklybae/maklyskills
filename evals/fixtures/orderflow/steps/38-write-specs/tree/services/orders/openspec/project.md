# orders

The service that owns an order from the moment a customer places it until it is
delivered or cancelled. Stock belongs to the inventory service; orders only ask
it to hold items and to give them back.

## Shape

Go, standard library only. Inside the service the layering is
`internal/api` → `internal/core` → `internal/store`, and it only ever points
that way: handlers translate HTTP, core holds every rule about what an order may
do next, stores know how an order is written down and nothing else.

Two backends exist behind the same store interface: a map for local work and a
single JSON document for anything that has to survive a restart.

## Conventions worth knowing before writing a spec

- Domain errors live in `internal/core/errors.go` and are what the API maps to
  status codes. A new failure a client can see needs a sentinel there.
- Anything the order flow asks of inventory goes through the `core.Inventory`
  interface, keyed by order id so a repeat is harmless.
- `AGENTS.md` in the repository root holds the rules a change is reviewed
  against. They are shorter than this file and they win.
