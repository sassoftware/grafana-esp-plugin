#!/usr/bin/env bash

set -e -o pipefail -o nounset

OAUTH_CLIENT_ID="${OAUTH_CLIENT_ID:-sv_client}"
export OAUTH_CLIENT_ID

ESP_NAMESPACE="${1}"

# Fetch access token to perform admin tasks:
function fetch_uaa_admin_token() {
    _resp=$(curl "https://${ESP_DOMAIN}/uaa/oauth/token" -s -k -X POST \
        -H 'Content-Type: application/x-www-form-urlencoded' \
        -H 'Accept: application/json' \
        -d "client_id=${UAA_ADMIN}&client_secret=${UAA_SECRET}&grant_type=client_credentials&response_type=token")

    echo "${_resp}" | jq -r '.access_token'
}

# Remove Grafana generic OAuth from allowed auth redirects:
function remove_grafana_auth_redirect() {
    _token="$(fetch_uaa_admin_token)"
    _redirect="https://${ESP_DOMAIN}/grafana/login/generic_oauth"

    _config=$(curl -s -k -X GET "https://${ESP_DOMAIN}/uaa/oauth/clients/${OAUTH_CLIENT_ID}" \
        -H "Authorization: Bearer ${_token}")

    _update_body=$(echo "${_config}" | jq -c -r --arg redirect "${_redirect}" \
        '.redirect_uri -= [$redirect] | {client_id: .client_id, redirect_uri: .redirect_uri}')

    _resp=$(curl "https://${ESP_DOMAIN}/uaa/oauth/clients/${OAUTH_CLIENT_ID}" -s -k -X PUT \
        -o /dev/null -w "%{http_code}" \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer ${_token}" \
        -H 'Accept: application/json' \
        -d "${_update_body}")

    if [ "${_resp}" == '200' ]; then
        echo "  Grafana OAuth redirect removed."
    else
        echo >&2 "WARN: Grafana OAuth client redirect removal failed with status code ${_resp}."
    fi
}

[ -d "./manifests" ] || {
    echo "No manifest directory found." >&2
    exit 1
}

[ -z "$KUBECONFIG" ] && {
    echo "KUBECONFIG environment variable unset." >&2
    exit 1
}

[ -z "${ESP_NAMESPACE}" ] && {
    echo "Usage: ${0} <esp-namespace>" >&2
    exit 1
}

echo "Fetching required deployment information..."
ESP_DOMAIN=$(kubectl -n "${ESP_NAMESPACE}" get ingress --output json |
    jq -r '.items[0].spec.rules[0].host')
export ESP_DOMAIN

_uaa_secret_data=$(kubectl -n "${ESP_NAMESPACE}" get secret uaa-secret --output json)
UAA_ADMIN=$(echo "${_uaa_secret_data}" | jq -r '.data.username | @base64d')
export UAA_ADMIN
UAA_SECRET=$(echo "${_uaa_secret_data}" | jq -r '.data.password | @base64d')
export UAA_SECRET

cat <<EOF
Deployment details:
  ESP domain:       ${ESP_DOMAIN}
  UAA admin user:   ${UAA_ADMIN}
  UAA admin secret: ****
  OAuth client ID:  ${OAUTH_CLIENT_ID}
EOF

echo "Removing Grafana from allowed OAuth client redirects..."
remove_grafana_auth_redirect
