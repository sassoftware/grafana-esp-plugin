#!/usr/bin/env bash

set -e -o pipefail -o nounset

function fetch_consul_token () {
    _token=$(kubectl -n "${ESP_NAMESPACE}" get secret sas-consul-client -o go-template='{{ .data.CONSUL_TOKEN | base64decode}}')

    echo ${_token}
}

function fetch_saslogon_token () {
    _token=$(fetch_consul_token)
    _resp=$(curl -k -X POST "https://$ESP_DOMAIN/SASLogon/oauth/clients/consul?callback=false&serviceId=app" -H "X-Consul-Token: ${_token}")

    echo "${_resp}" | jq -r '.access_token'
}

function register_oauth_client () {
    _token="$(fetch_saslogon_token)"

    _redirecturl="https://${ESP_DOMAIN}/grafana/login/generic_oauth"

    _body='{
        "scope": ["*"],
        "client_id": "'"${OAUTH_CLIENT_ID}"'",
        "client_secret": "'"${OAUTH_CLIENT_SECRET}"'",
        "authorized_grant_types": ["authorization_code"],
        "redirect_uri": ["'"${_redirecturl}"'"],
        "autoapprove": ["true"],
        "name": "Grafana"
    }'

    _resp=$(curl -k -X POST "https://$ESP_DOMAIN/SASLogon/oauth/clients" \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer ${_token}" \
        -d "${_body}")

    regex_error="error"
    if [[ "${_resp}" =~ $regex_error ]]; then
       error=$(echo "${_resp}" | jq -r '.error')
       error_description=$(echo "${_resp}" | jq -r '.error_description')
       echo >&2 "Failed to register Grafana as OAuth client"
       echo >&2 "${error}: ${error_description}"

    else
       echo "Grafana registered as OAuth client"
    fi

}

register_oauth_client
