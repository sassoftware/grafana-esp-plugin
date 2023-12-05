#!/usr/bin/env bash

set -e -o pipefail -o nounset

ESP_NAMESPACE="${1}"
GRAFANA_NAMESPACE="${2:-${ESP_NAMESPACE}}"
OAUTH_CLIENT_ID="${OAUTH_CLIENT_ID:-sv_client}"; export OAUTH_CLIENT_ID
OAUTH_CLIENT_SECRET="${OAUTH_CLIENT_SECRET:-secret}"; export OAUTH_CLIENT_SECRET

function usage () {
    echo "Usage: ${0} <esp-namespace> " >&2
    exit 1
}

[ -z "$KUBECONFIG" ] && {
    echo "KUBECONFIG environment variable unset." >&2
    exit 1
}

[ -z "${ESP_NAMESPACE}" ] && {
    usage
}

ESP_DOMAIN=$(kubectl -n "${ESP_NAMESPACE}" get ingress --output json | jq -r '.items[0].spec.rules[0].host')
GRAFANA_DOMAIN=$(kubectl -n "${GRAFANA_NAMESPACE}" get ingress --output json | jq -r '.items[0].spec.rules[0].host')

# Fetch access token to perform admin tasks:
function fetch_uaa_admin_token() {
    _resp=$(curl "https://${ESP_DOMAIN}/uaa/oauth/token" -k -X POST \
        -H 'Content-Type: application/x-www-form-urlencoded' \
        -H 'Accept: application/json' \
        -d "client_id=${UAA_ADMIN}&client_secret=${UAA_SECRET}&grant_type=client_credentials&response_type=token")

    echo "${_resp}" | jq -r '.access_token'
}

# Add Grafana generic OAuth to allowed auth redirects:
function add_grafana_auth_redirect_uaa() {
    _token="$(fetch_uaa_admin_token)"
    _redirect="https://${GRAFANA_DOMAIN}/grafana/login/generic_oauth"

    _config=$(curl -k -X GET "https://${ESP_DOMAIN}/uaa/oauth/clients/${OAUTH_CLIENT_ID}" -H "Authorization: Bearer ${_token}")

    _update_body=$(echo "${_config}" | jq -c -r --arg redirect "${_redirect}" \
        '.redirect_uri += [$redirect] | {client_id: .client_id, redirect_uri: .redirect_uri}')

    _resp=$(curl "https://${ESP_DOMAIN}/uaa/oauth/clients/${OAUTH_CLIENT_ID}" -k -X PUT \
        -o /dev/null -w "%{http_code}" \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer ${_token}" \
        -H 'Accept: application/json' \
        -d "${_update_body}")

    if [ "${_resp}" == '200' ]; then
        echo "  Grafana OAuth redirect added."
    else
        echo >&2 "ERROR: OAuth client redirect update failed with status code ${_resp}."
        exit 1
    fi
}

_uaa_secret_data=$(kubectl -n "${ESP_NAMESPACE}" get secret uaa-secret --output json)
UAA_ADMIN=$(echo "${_uaa_secret_data}" | jq -r '.data.username | @base64d')
export UAA_ADMIN
UAA_SECRET=$(echo "${_uaa_secret_data}" | jq -r '.data.password | @base64d')
export UAA_SECRET

cat <<EOF
OAuth details:
  ESP Domain:         ${ESP_DOMAIN}
  Grafana Domain:      ${GRAFANA_DOMAIN}
  OAuth client ID:     ${OAUTH_CLIENT_ID}
  OAuth client secret: ${OAUTH_CLIENT_SECRET}
  UAA Admin:     ${UAA_ADMIN}
  UAA secret: ${UAA_SECRET}
EOF

add_grafana_auth_redirect_uaa
