#!/bin/sh
set -eu

: "${API_TOKEN:?set API_TOKEN to the public fixture from .env.example}"
base_url="${BASE_URL:-http://127.0.0.1:8080}"
response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT HUP INT TERM

status="$(curl --silent --show-error \
  --connect-timeout 5 \
  --max-time 10 \
  --output "$response_file" \
  --write-out '%{http_code}' \
  --request POST \
  --header "Authorization: Bearer ${API_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data-binary '{"sku":"DEMO-WIDGET","quantity":2,"customer_ref":"cust_demo_1001"}' \
  "${base_url}/v1/orders")"

if [ "$status" != "201" ]; then
  printf 'unexpected HTTP status %s\n' "$status" >&2
  sed -n '1,20p' "$response_file" >&2
  exit 1
fi

sed -n '1,20p' "$response_file"
