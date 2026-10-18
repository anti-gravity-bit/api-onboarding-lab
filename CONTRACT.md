# Frozen contract

This document is the behavior the tests and the OpenAPI file must match. Change it only when Grok updates the tests and the spec in the same change.

## Routes

| Method and path | Auth |
| --- | --- |
| `GET /healthz` | none |
| `POST /v1/orders` | `Authorization: Bearer <API_TOKEN>` |
| `GET /v1/orders/{id}` | bearer token |
| `POST /v1/webhooks/order` | `Lab-Signature` HMAC |

No other route is part of the demo. Unmatched paths return 404 JSON `not_found`.

## Orders

`POST /v1/orders` body, unknown fields rejected:

```json
{"sku":"DEMO-WIDGET","quantity":2,"customer_ref":"cust_demo_1001"}
```

- `sku`: required, `^[A-Z0-9][A-Z0-9_-]{0,63}$`.
- `quantity`: required JSON integer from 1 to 100. Strings and fractions fail validation.
- `customer_ref`: required, `^[A-Za-z0-9_-]{1,64}$`.

Response 201:

```json
{"id":"ord_0123456789abcdef","sku":"DEMO-WIDGET","quantity":2,"customer_ref":"cust_demo_1001","status":"accepted","created_at":"2026-10-19T00:00:00Z"}
```

`created_at` is UTC RFC3339. The store keeps the record until the process exits. Nothing is written to disk.

## Webhook

Header, with no spaces:

```text
Lab-Signature: t=1710000000,v1=<64 lowercase hex chars>
```

Signed bytes are the decimal timestamp, a single `.`, then the raw request body. The MAC is HMAC-SHA256 over those bytes, keyed with `WEBHOOK_SECRET`. Compare the decoded MAC with `hmac.Equal`. Hex-decode and length checks happen before the compare. A failed decode is a malformed signature.

The clock is injectable. Acceptance is `abs(nowUnix - t) <= WEBHOOK_TOLERANCE_SECONDS`. Default tolerance is 300.

After the MAC matches, decode JSON and reject unknown fields:

```json
{"id":"evt_demo_1001","type":"order.accepted","order_id":"ord_0123456789abcdef","occurred_at":"2026-10-19T00:00:00Z"}
```

- `id`: `^[A-Za-z0-9_-]{1,80}$`.
- `type`: the only accepted value is `order.accepted`.
- `order_id`: `^ord_[0-9a-f]{16}$`. The receiver does not check that this order was created here.
- `occurred_at`: RFC3339.

First success stores the event id in memory and returns 202:

```json
{"status":"accepted","duplicate":false,"event_id":"evt_demo_1001"}
```

A later valid event with the same id returns 200:

```json
{"status":"duplicate","duplicate":true,"event_id":"evt_demo_1001"}
```

The check-and-store is serialized so two concurrent first deliveries produce one 202 and one 200. Failed auth or schema checks do not store the id.

## Errors

`Content-Type` is `application/json`. Codes:

| HTTP | `error.code` |
| --- | --- |
| 400 malformed JSON | `bad_request` |
| 400 schema | `validation_error` |
| 401 bad token or signature | `unauthorized` |
| 404 | `not_found` |
| 405 | `method_not_allowed` |
| 413 | `payload_too_large` |
| 415 | `unsupported_media_type` |

Field failures use the message `request validation failed` and a `details` array of `{field, message}`. Other errors omit `details`. Messages must not contain the token, the webhook secret, or the request body.

## Limits and timeouts

- Max body: 65536 bytes for order and webhook posts.
- `ReadHeaderTimeout` 5s, `ReadTimeout` 10s, `WriteTimeout` 15s, `IdleTimeout` 60s.
- Graceful shutdown waits up to `SHUTDOWN_TIMEOUT`, default 10s, then returns that context error.

## Environment

| Name | Default | Rule |
| --- | --- | --- |
| `ADDR` | `127.0.0.1:8080` | Host demo binds loopback unless this is set |
| `API_TOKEN` | none | Required, at least 32 bytes |
| `WEBHOOK_SECRET` | none | Required, at least 32 bytes |
| `WEBHOOK_TOLERANCE_SECONDS` | `300` | Inclusive skew window |
| `MAX_BODY_BYTES` | `65536` | |
| `SHUTDOWN_TIMEOUT` | `10s` | Go duration |

The container image may set `ADDR=:8080`. It must not set the token or the secret.

## Package layout

- `cmd/labd` starts the process.
- `internal/config` reads the environment.
- `internal/httpapi` is the router, errors, and handlers.
- `internal/store` is the in-memory order and event store.
- `webhooksig` is the public signer and verifier used by the server and the Go example.

Handlers take a clock function so tests do not depend on wall time. Tests use the fixture values from `.env.example` and do not read a developer `.env`.
