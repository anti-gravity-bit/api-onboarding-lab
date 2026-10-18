# API Onboarding Lab

A local Go demo of a synthetic order API and a signed webhook receiver. It is a personal portfolio project for backend engineers who need to show API onboarding and integration tests. Orders and webhook event ids stay in process memory and disappear when the process exits.

The checked-in token and webhook secret are public fixtures. This repository does not deploy the API.

## Quickstart

These commands were run on 2026-10-19 with `go1.27.2`. The transcript is `evidence/quickstart.txt`.

```bash
export API_TOKEN=demo-api-token-not-for-production-use-32b
export WEBHOOK_SECRET=demo-webhook-secret-not-for-production-use
export ADDR=127.0.0.1:8080
go run ./cmd/labd
```

In a second shell, with the same two exports and `BASE_URL=http://127.0.0.1:8080`:

```bash
curl --silent --show-error "$BASE_URL/healthz"
sh examples/curl/create-order.sh
sh examples/curl/send-webhook.sh
sh examples/curl/send-webhook.sh
go run ./examples/go-client
```

The second webhook call for the same event returns HTTP 200 and `"duplicate": true`.

## What is included

- `GET /healthz`, `POST /v1/orders`, `GET /v1/orders/{id}`, and `POST /v1/webhooks/order`.
- JSON errors, request ids, body size limits, timeouts, and graceful shutdown.
- HMAC-SHA256 webhook signatures over the raw body, a 300 second timestamp window, and in-memory duplicate detection.
- A Go client, curl scripts, OpenAPI 3.0.3, a multi-stage non-root Dockerfile, and loopback-only Compose.
- Tests for malformed input, rejected signatures, replay outside the time window, and concurrent duplicate delivery.

The behavior contract is [CONTRACT.md](CONTRACT.md). The spec is [api/openapi.yaml](api/openapi.yaml). A documentation website is not published yet.

## Tests

`go test -count=1 ./...` and `go test -count=1 -race ./...` passed with Go 1.27.2. Transcripts are in [evidence/](evidence/). `staticcheck` 2026.2.2, `golangci-lint` 2.15.0, and `govulncheck` v1.9.0 reported no findings on this tree. Those binaries were installed for the check. They are not pinned in CI yet, and there is no GitHub Actions run to badge.

## License and independence

Licensed under the MIT License. Copyright 2026 Abir Saarkar. This is independent personal work and uses synthetic records only.

Report security issues through the process in SECURITY.md once that file is added. Until then, use the contact on the GitHub profile for `anti-gravity-bit` and do not open a public issue for an unfixed vulnerability.
