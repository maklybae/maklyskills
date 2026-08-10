# Rules for changes in this repository

Two rules, both of them load bearing. A review that finds one of them broken
sends the change back, so please read them before the first commit rather than
after.

## 1. Wrap errors with `pkg/xerrors`

Every error that crosses a package boundary gets context:
`xerrors.Wrap(err, "load order")` or `xerrors.Wrapf(err, "load order %s", id)`.
Bare `fmt.Errorf` is not allowed — it produced messages that told us an order id
was missing without telling us which call wanted it, and it made `errors.Is`
depend on whether whoever wrote the line remembered `%w`. New sentinel values
still come from `errors.New`, in the `errors.go` of the service that owns them.

## 2. Handlers stay thin

`internal/api` decodes the request, calls exactly one `internal/core` method and
turns the result into a response or a status code. The package does not import
`internal/store` — its tests sit in `package api_test` and may wire up whatever
they need — and it does not decide what an order may do next: a handler that
needs a new decision needs a new core method instead. The mapping from domain
errors to HTTP status codes is the one piece of policy the API layer owns.
