#!/usr/bin/env bash

set -e -o pipefail -o nounset

ESP_NAMESPACE="${1}"
KEYCLOAK_SUBPATH="${KEYCLOAK_SUBPATH:-auth}"

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

function check_keycloak_deployment() {
    if ! kubectl -n "${ESP_NAMESPACE}" get deployment keycloak-deployment 2>/dev/null 1>&2; then
        echo >&2 "ERROR: No Keycloak deployment found under namespace ${ESP_NAMESPACE}."
        exit 1
    fi

    _kc_pod=$(kubectl -n "${ESP_NAMESPACE}" get pods -o json |
        jq -r '.items[] | select(.metadata.name | test("^keycloak-deployment-")) | .metadata.name')
    [ -n "${_kc_pod}" ] || {
        echo >&2 "ERROR: No keycloak-deployment-* pod found under namespace ${ESP_NAMESPACE}."
        exit 1
    }

    _kc_ready=$(kubectl -n "${ESP_NAMESPACE}" get pod "${_kc_pod}" -o json |
        jq -r '.status.conditions[] | select(.type == "Ready") | .status')
    [ "${_kc_ready}" == 'True' ] || {
        echo >&2 "ERROR: Keycloak deployment exists but is not ready. Try again later."
        exit 1
    }
}

function check_requirements() {

    if ! kubectl -n "${ESP_NAMESPACE}" get secret keycloak-admin-secret 2>/dev/null 1>&2; then
        echo >&2 "ERROR: No Keycloak admin secret found under namespace ${ESP_NAMESPACE}."
        exit 1
    fi

    if ! kubectl -n "${ESP_NAMESPACE}" get secret oauth2-proxy-client-secret 2>/dev/null 1>&2; then
        echo >&2 "ERROR: No OAuth2 Proxy client secret found under namespace ${ESP_NAMESPACE}."
        exit 1
    fi

    check_keycloak_deployment
}

# Fetch access token to perform admin tasks:
function fetch_keycloak_admin_token() {
    _resp=$(curl "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/realms/master/protocol/openid-connect/token" -k -X POST \
        -H 'Content-Type: application/x-www-form-urlencoded' \
        -H 'Accept: application/json' \
        -d "client_id=admin-cli&grant_type=password&username=${KEYCLOAK_ADMIN}&password=${KEYCLOAK_SECRET}")

    echo "${_resp}" | jq -r '.access_token'
}

function create_role() {
    _role_name="${1}"
    _role_repr="{\"name\": \"${_role_name}\", \"clientRole\": true}"
    curl "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients/${_client_id}/roles" -k -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${_token}" \
        -d "${_role_repr}"
}

function add_protocol_mapper() {
    _mapper_repr=$(echo -e "
    {
     \"name\": \"GrafanaRoles\",
     \"protocol\": \"openid-connect\",
     \"protocolMapper\": \"oidc-usermodel-client-role-mapper\",
     \"consentRequired\": false,
     \"config\": {
       \"claim.name\": \"grafana_roles\",
       \"usermodel.clientRoleMapping.clientId\": \"${OAUTH_CLIENT_ID}\",
       \"jsonType.label\": \"String\",
       \"multivalued\": \"true\",
       \"access.token.claim\": \"true\",
       \"userinfo.token.claim\": \"false\",
       \"id.token.claim\": \"true\"
       }
    }")
    _mapper_body=$(echo "${_mapper_repr}" | jq -r -c)
    curl -k -X POST \
        "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients/${_client_id}/protocol-mappers/models" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${_token}" \
        -d "${_mapper_body}"
}

function prepare_keycloak_roles() {
    _token="$(fetch_keycloak_admin_token)"
    # Get sas-esp realm clients:
    _kc_clients=$(curl -k -X GET "https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/admin/realms/sas-esp/clients" -H "Authorization: Bearer ${_token}")
    # Get OAuth2 Proxy client ID:
    _client_id=$(echo "${_kc_clients}" | jq -r --arg opid "${OAUTH_CLIENT_ID}" '.[] | select(.clientId == $opid) | .id')
    # Create Grafana roles:
    create_role "grafana-admin"
    create_role "admin"
    create_role "editor"
    # Create Grafana role protocol mapper:
    add_protocol_mapper
}

_keycloak_admin_secret=$(kubectl -n "${ESP_NAMESPACE}" get secret keycloak-admin-secret --output json)
KEYCLOAK_ADMIN=$(echo "${_keycloak_admin_secret}" | jq -r '.data.username | @base64d')
KEYCLOAK_SECRET=$(echo "${_keycloak_admin_secret}" | jq -r '.data.password | @base64d')

_oauth2_proxy_secret=$(kubectl -n "${ESP_NAMESPACE}" get secret oauth2-proxy-client-secret --output json)
OAUTH_CLIENT_ID=$(echo "${_oauth2_proxy_secret}" | jq -r '.data.OAUTH2_PROXY_CLIENT_ID | @base64d')
export OAUTH_CLIENT_ID
OAUTH_CLIENT_SECRET=$(echo "${_oauth2_proxy_secret}" | jq -r '.data.OAUTH2_PROXY_CLIENT_SECRET | @base64d')
export OAUTH_CLIENT_SECRET

cat <<EOF
OAuth details:
  ESP Domain:         ${ESP_DOMAIN}
  OAuth client ID:     ${OAUTH_CLIENT_ID}
  OAuth client secret: ${OAUTH_CLIENT_SECRET}
EOF

prepare_keycloak_roles
