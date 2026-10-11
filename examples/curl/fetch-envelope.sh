#!/bin/sh
set -eu

: "${ENVVAULT_TOKEN:?set ENVVAULT_TOKEN}"
: "${ENVVAULT_VAULT:?set ENVVAULT_VAULT}"
: "${ENVVAULT_SECRET:?set ENVVAULT_SECRET}"

base_url=${ENVVAULT_URL:-https://www.envvault.net}

# This prints encrypted JSON, never plaintext. The key is intentionally unused.
curl --fail-with-body --silent --show-error \
  --header "Authorization: Bearer ${ENVVAULT_TOKEN}" \
  --header 'Accept: application/json' \
  "${base_url%/}/v1/api/${ENVVAULT_VAULT}/${ENVVAULT_SECRET}"
