#!/bin/bash

set -e
set -o pipefail

export TKN=$(curl -X POST $KEYCLOAK_URL/auth/realms/master/protocol/openid-connect/token \
 --fail \
 -H "Content-Type: application/x-www-form-urlencoded" \
 -d "username=sepl" \
 -d 'password=sepl' \
 -d 'grant_type=password' \
 -d 'client_id=admin-cli' | jq -r '.access_token')

curl -X POST --data "@./realm-export.json" --silent --show-error --fail $KEYCLOAK_URL/auth/admin/realms/master/partialImport \
-H "Content-Type: application/json" \
-H "Accept: application/json" \
-H "Authorization: Bearer $TKN"
