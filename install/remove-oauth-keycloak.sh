#!/usr/bin/env bash

set -e -o pipefail -o nounset

KEYCLOAK_SUBPATH="${KEYCLOAK_SUBPATH:-auth}"

ESP_NAMESPACE="${1}"

function check_requirements() {

  [ -z "$KUBECONFIG" ] && {
    echo "KUBECONFIG environment variable unset." >&2
    exit 1
  }

  [ -z "${ESP_NAMESPACE}" ] && {
    echo "Usage: ${0} <esp-namespace>" >&2
    exit 1
  }

  if ! kubectl get namespace "${ESP_NAMESPACE}" 2>/dev/null 1>&2; then
    echo >&2 "ERROR: Namespace ${ESP_NAMESPACE} not found."
    exit 1
  fi

  if ! kubectl -n "${ESP_NAMESPACE}" get secret keycloak-admin-secret 2>/dev/null 1>&2; then
    echo >&2 "ERROR: No Keycloak admin secret found under namespace ${ESP_NAMESPACE}."
    exit 1
  fi

  if ! kubectl -n "${ESP_NAMESPACE}" get secret oauth2-proxy-client-secret 2>/dev/null 1>&2; then
    echo >&2 "ERROR: No OAuth2 Proxy client secret found under namespace ${ESP_NAMESPACE}."
    exit 1
  fi
}

# Fetch access token to perform admin tasks:
function fetch_keycloak_admin_token() {
  _resp=$(curl "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/realms/master/protocol/openid-connect/token" -s -k -X POST \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -H 'Accept: application/json' \
    -d "client_id=admin-cli&grant_type=password&username=${KEYCLOAK_ADMIN}&password=${KEYCLOAK_SECRET}")

  echo "${_resp}" | jq -r '.access_token'
}

function delete_role() {
  _role_name="${1}"
  curl -s -k -X DELETE \
    "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients/${_client_id}/roles/${_role_name}" \
    -H "Authorization: Bearer ${_token}"
}

function remove_protocol_mapper() {
  # Get mapper id:
  _mappers=$(curl -s -k -X GET "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients/${_client_id}/protocol-mappers/models" -H "Authorization: Bearer ${_token}")
  _mapper_id=$(echo "${_mappers}" | jq -r '.[] | select(.name == "GrafanaRoles") | .id')
  # Delete mapper:
  curl -s -k -X DELETE "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients/${_client_id}/protocol-mappers/models/${_mapper_id}" -H "Authorization: Bearer ${_token}"
}

function remove_keycloak_roles() {
  _token="$(fetch_keycloak_admin_token)"
  # Get sas-esp realm clients:
  _kc_clients=$(curl -s -k -X GET "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients" -H "Authorization: Bearer ${_token}")
  # Get OAuth2 Proxy client ID:
  _client_id=$(echo "${_kc_clients}" | jq -r --arg opid "${OAUTH_CLIENT_ID}" '.[] | select(.clientId == $opid) | .id')
  # Delete Grafana roles:
  delete_role "grafana-admin"
  delete_role "admin"
  delete_role "editor"
  # Remove Grafana role protocol mapper:
  remove_protocol_mapper
}

# Fail fast on missing requirements:
check_requirements

echo "Fetching required deployment information..."
ESP_DOMAIN=$(kubectl -n "${ESP_NAMESPACE}" get ingress --output json | jq -r '.items[0].spec.rules[0].host')
export ESP_DOMAIN

_oauth2_proxy_secret=$(kubectl -n "${ESP_NAMESPACE}" get secret oauth2-proxy-client-secret --output json)
OAUTH_CLIENT_ID=$(echo "${_oauth2_proxy_secret}" | jq -r '.data.OAUTH2_PROXY_CLIENT_ID | @base64d')
export OAUTH_CLIENT_ID

_keycloak_admin_secret=$(kubectl -n "${ESP_NAMESPACE}" get secret keycloak-admin-secret --output json)
KEYCLOAK_ADMIN=$(echo "${_keycloak_admin_secret}" | jq -r '.data.username | @base64d')
export KEYCLOAK_ADMIN
KEYCLOAK_SECRET=$(echo "${_keycloak_admin_secret}" | jq -r '.data.password | @base64d')
export KEYCLOAK_SECRET

cat <<EOF
Deployment details:
  ESP domain:            ${ESP_DOMAIN}
  Keycloak admin user:   ${KEYCLOAK_ADMIN}
  Keycloak admin secret: ****
  OAuth client ID:       ${OAUTH_CLIENT_ID}
EOF

echo "Removing Grafana roles and mapper from Keycloak client..."
remove_keycloak_roles
