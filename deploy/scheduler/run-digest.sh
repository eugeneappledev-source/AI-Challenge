#!/bin/sh
set -eu

curl --fail --silent --show-error \
  --request POST "${RADAR_API_URL}/v1/radar/digests/run" \
  --header "Authorization: Bearer ${APP_ACCESS_TOKEN}" \
  --header "Content-Type: application/json" \
  --data '{"trigger":"cron"}'
