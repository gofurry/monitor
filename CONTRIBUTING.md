# Contributing

Thanks for helping improve `monitor`.

This project is intentionally small: one `net/http` middleware, one status page, and one JSON snapshot. Please keep changes focused on that scope.

## Development

Requirements:

- Go 1.24 or newer

Before opening a pull request, run:

```sh
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
```

For concurrency-sensitive changes, also run:

```sh
go test -race ./...
```

## Design Guidelines

- Keep the public API small and easy to use.
- Prefer the Go standard library unless a dependency has clear value.
- Do not add frontend build tooling or external browser assets.
- Do not count monitor endpoint requests as business traffic.
- Keep request-path overhead low.
- Document concurrency behavior for reusable exported types.
- Return useful zero values when optional metric collection fails.

## Pull Requests

Please include:

- a short description of the change
- tests for behavior changes
- documentation updates when public behavior or configuration changes

Avoid mixing unrelated refactors with feature or bug-fix changes.
