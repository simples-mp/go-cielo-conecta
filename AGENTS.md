# Repository Guidelines

## Project Structure & Module Organization

This Go library targets the Cielo Conecta API. Production code lives in the repository root under package `go_cielo_conecta`. Core request and authentication behavior is in `client.go`; payment, cancellation, and sale workflows are split across `payment.go`, `cancellation.go`, and `sale_handler.go`. Shared API models and environment values are defined in `types.go` and `vars.go`. Enum string mappings use the `*_strings.go` naming pattern. Integration tests live in `tests/` and use the external `tests` package to exercise the public API. There are no bundled assets.

## Build, Test, and Development Commands

- `go build ./...` — compile every package without producing a checked-in binary.
- `go test ./...` — run all tests; credential-dependent tests skip when variables are absent.
- `MERCHANT_ID=... MERCHANT_SECRET=... go test ./tests -v` — run the Cielo sandbox/homologation integration tests.
- `go test -race ./...` — check concurrent code paths for data races.
- `go vet ./...` — run Go's standard static analysis.
- `gofmt -w *.go tests/*.go` — format source and tests before committing.
- `go mod tidy` — reconcile `go.mod` and `go.sum` after dependency changes.

## Coding Style & Naming Conventions

Follow standard Go conventions and let `gofmt` control tabs and layout. Use PascalCase for exported API types and methods, camelCase for internal identifiers, and initialisms such as `ID`, `URL`, and `HTTP` consistently. Keep files focused by API domain. Accept `context.Context` on network operations and wrap errors with useful operation context. Preserve backward compatibility for exported interfaces and structs unless a breaking change is intentional and documented.

## Testing Guidelines

Use Go's `testing` package. Name tests `TestXxx` in `*_test.go` files. Prefer table-driven unit tests and `httptest` for deterministic HTTP behavior; reserve live API calls for `tests/`. Never hard-code merchant credentials. Add regression coverage for error responses, JSON encoding/decoding, and environment-specific headers.

## Commit & Pull Request Guidelines

Recent commits use short, imperative summaries such as `Add detailed logging...` and `Refactor logging...`; follow that pattern and keep each commit focused. Pull requests should explain behavior changes, list validation commands, link relevant issues, and call out API compatibility or credential/configuration impacts. Include sanitized request/response examples when they clarify an integration change; screenshots are generally unnecessary for this library.

## Security & Configuration

Keep `.env`, tokens, merchant IDs, secrets, and payment-card data out of commits and logs. Redact sensitive headers and payload fields in tests, examples, and debugging output.
