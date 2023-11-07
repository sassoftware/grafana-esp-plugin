#!/usr/bin/env bash

set -e -o pipefail -o nounset

OAUTH_CLIENT_ID="${OAUTH_CLIENT_ID:-sv_client}"
export OAUTH_CLIENT_ID

OAUTH_CLIENT_SECRET="${OAUTH_CLIENT_SECRET:-secret}"
export OAUTH_CLIENT_SECRET

ESP_NAMESPACE="${1}"

ESP_PLUGIN_SOURCE="${2}"
export ESP_PLUGIN_SOURCE

DRYRUN="${3}"

# Fetch access token to perform admin tasks:
function fetch_uaa_admin_token() {
    _resp=$(curl "https://${ESP_DOMAIN}/uaa/oauth/token" -s -k -X POST \
        -H 'Content-Type: application/x-www-form-urlencoded' \
        -H 'Accept: application/json' \
        -d "client_id=${UAA_ADMIN}&client_secret=${UAA_SECRET}&grant_type=client_credentials&response_type=token")

    echo "${_resp}" | jq -r '.access_token'
}

# Add Grafana generic OAuth to allowed auth redirects:
function add_grafana_auth_redirect() {
    _token="$(fetch_uaa_admin_token)"
    _redirect="https://${ESP_DOMAIN}/grafana/login/generic_oauth"

    _config=$(curl -s -k -X GET "https://${ESP_DOMAIN}/uaa/oauth/clients/${OAUTH_CLIENT_ID}" -H "Authorization: Bearer ${_token}")

    _update_body=$(echo "${_config}" | jq -c -r --arg redirect "${_redirect}" \
        '.redirect_uri += [$redirect] | {client_id: .client_id, redirect_uri: .redirect_uri}')

    _resp=$(curl "https://${ESP_DOMAIN}/uaa/oauth/clients/${OAUTH_CLIENT_ID}" -s -k -X PUT \
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

[ -z "$KUBECONFIG" ] && {
    echo "KUBECONFIG environment variable unset." >&2
    exit 1
}

[ -z "${ESP_NAMESPACE}" ] && {
    echo "Usage: ${0} <namespace> <plugin-zip-url>" >&2
    exit 1
}

[ -z "${ESP_PLUGIN_SOURCE}" ] && {
    echo "Usage: ${0} <namespace> <plugin-zip-url>" >&2
    exit 1
}

if [ -d "./manifests" ]; then
    echo "Existing manifest directory found." >&2
    echo "Removing manifests..."
    rm -r ./manifests/
fi

echo "Fetching required deployment information..."
ESP_DOMAIN=$(kubectl -n "${ESP_NAMESPACE}" get ingress --output json | jq -r '.items[0].spec.rules[0].host')
export ESP_DOMAIN

_uaa_secret_data=$(kubectl -n "${ESP_NAMESPACE}" get secret uaa-secret --output json)
UAA_ADMIN=$(echo "${_uaa_secret_data}" | jq -r '.data.username | @base64d')
export UAA_ADMIN
UAA_SECRET=$(echo "${_uaa_secret_data}" | jq -r '.data.password | @base64d')
export UAA_SECRET

cat <<EOF
Deployment details:
  ESP domain:          ${ESP_DOMAIN}
  UAA admin user:      ${UAA_ADMIN}
  UAA admin secret:    ****
  OAuth client ID:     ${OAUTH_CLIENT_ID}
  OAuth client secret: ****
Deploying Grafana with values:
  ESP plugin source:   ${ESP_PLUGIN_SOURCE}
EOF

echo "Adding Grafana to allowed OAuth client redirects..."
add_grafana_auth_redirect

echo "Generating manifests..."
[ -d "./manifests" ] || mkdir "manifests"
cp -r *.yaml manifests/

find ./manifests/ -type f -name "*.yaml" -exec perl -pi -e 's/\QTEMPLATE_ESP_DOMAIN/$ENV{"ESP_DOMAIN"}/g' '{}' +
find ./manifests/ -type f -name "*.yaml" -exec perl -pi -e 's/\QTEMPLATE_OAUTH_CLIENT_ID/$ENV{"OAUTH_CLIENT_ID"}/g' '{}' +
find ./manifests/ -type f -name "*.yaml" -exec perl -pi -e 's/\QTEMPLATE_OAUTH_CLIENT_SECRET/$ENV{"OAUTH_CLIENT_SECRET"}/g' '{}' +
find ./manifests/ -type f -name "*.yaml" -exec perl -pi -e 's/\QTEMPLATE_ESP_PLUGIN_SOURCE/$ENV{"ESP_PLUGIN_SOURCE"}/g' '{}' +

if [[ -z "${DRYRUN}" ]]; then

  if [[ "${INSTALL_GRAFANA}" ]]; then
    echo "Installing grafana"

    kubectl -n "${ESP_NAMESPACE}" apply -f ./manifests/grafana.yaml
  fi

  echo "Applying config-map.yaml"

  kubectl -n "${ESP_NAMESPACE}" apply -f ./manifests/config-map.yaml

  echo "Patching deployment/grafana with patch-grafana.yaml"

  kubectl -n "${ESP_NAMESPACE}" patch --patch-file ./manifests/patch-grafana.yaml deployment/grafana

fi

if [[ "${DRYRUN}" ]]; then
  echo "Dry run specified. Printing manifests to be applied:"
  less ./manifests/config-map.yaml
  less ./manifests/patch-grafana.yaml
fi
