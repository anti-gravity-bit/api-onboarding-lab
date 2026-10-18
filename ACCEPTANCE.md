# Acceptance matrix

A row is accepted only when a test or an executed command shows it. Nothing in this file has been executed yet.

| ID | Behavior | Expected |
| --- | --- | --- |
| A1 | `GET /healthz` with no auth | 200 `{"status":"ok"}` |
| A2 | `POST /v1/orders` with a valid bearer token and body | 201 order, `status` `accepted`, id `ord_` plus 16 lowercase hex digits |
| A3 | `GET /v1/orders/{id}` for that order | 200 same order |
| A4 | `GET /v1/orders/{id}` well-formed and unknown | 404 `not_found` |
| A5 | Order id with the wrong shape | 400 `validation_error` after auth succeeds |
| A6 | Missing or wrong bearer token on either order route | 401 `unauthorized`. Same message for missing and wrong |
| A7 | Order body is not JSON | 400 `bad_request` |
| A8 | Unknown JSON field, missing field, bad sku, bad customer_ref, quantity out of range, or a non-integer quantity | 400 `validation_error` |
| A9 | Body larger than 65536 bytes | 413 `payload_too_large` |
| A10 | Content-Type other than `application/json` or `application/json; charset=utf-8` | 415 `unsupported_media_type` |
| A11 | Valid `Lab-Signature` webhook | 202 `duplicate` false |
| A12 | Malformed, missing, or non-matching signature | 401 `unauthorized` and the event is not stored |
| A13 | Timestamp older or newer than 300 seconds | 401 and the event is not stored. A skew of exactly 300 seconds is accepted |
| A14 | Second valid delivery of the same event id | 200 `duplicate` true |
| A15 | Two concurrent valid deliveries of one new event id | One 202 and one 200. One stored event |
| A16 | Valid signature and invalid JSON or schema | 400 and the event is not stored |
| A17 | Inbound `X-Request-Id` of 1 to 64 characters in `[A-Za-z0-9._-]` | Echoed in the response header and error body. Otherwise a generated `req_` id |
| A18 | Process-local restart, modeled by a new store | Prior orders and event ids are gone |
| A19 | `Shutdown` on the HTTP server | Returns after in-flight work and then refuses new requests |
| A20 | Startup with missing or short `API_TOKEN` or `WEBHOOK_SECRET` | Non-zero exit. The value is not printed |

Error JSON is always:

```json
{"error":{"code":"validation_error","message":"request validation failed","request_id":"req_0123456789abcdef","details":[{"field":"quantity","message":"must be an integer from 1 to 100"}]}}
```

`details` is present only for field validation.
