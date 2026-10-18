#!/bin/sh
set -eu

: "${WEBHOOK_SECRET:?set WEBHOOK_SECRET to the public fixture from .env.example}"
base_url="${BASE_URL:-http://127.0.0.1:8080}"
timestamp="$(date +%s)"
body='{"id":"evt_curl_demo_1001","type":"order.accepted","order_id":"ord_0123456789abcdef","occurred_at":"2026-10-19T00:00:00Z"}'
signature="$(printf '%s' "$body" | go run ./examples/curl/signature "$timestamp")"
response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT HUP INT TERM

status="$(curl --silent --show-error \
  --connect-timeout 5 \
  --max-time 10 \
  --output "$response_file" \
  --write-out '%{http_code}' \
  --request POST \
  --header 'Content-Type: application/json' \
  --header "Lab-Signature: ${signature}" \
  --data-binary "$body" \
  "${base_url}/v1/webhooks/order")"

case "$status" in
  200|202) ;;
  *)
    printf 'unexpected HTTP status %s\n' "$status" >&2
    sed -n '1,20p' "$response_file" >&2
    exit 1
    ;;
esac

sed -n '1,20p' "$response_file"
