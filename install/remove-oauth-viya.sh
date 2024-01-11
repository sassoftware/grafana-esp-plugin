#!/usr/bin/env bash

set -e -o pipefail -o nounset

ESP_NAMESPACE="${1}"
GRAFANA_NAMESPACE="${2:-${ESP_NAMESPACE}}"
OAUTH_CLIENT_ID="${OAUTH_CLIENT_ID:-sv_client}"; export OAUTH_CLIENT_ID
OAUTH_CLIENT_SECRET="${OAUTH_CLIENT_SECRET:-secret}"; export OAUTH_CLIENT_SECRET

function usage () {
    echo "Usage: ${0} <viya-namespace> <grafana-namespace>" >&2
    exit 1
}

[ -z "${KUBECONFIG-}" ] && {
    echo "KUBECONFIG environment variable unset." >&2
    exit 1
}

[ -z "${ESP_NAMESPACE-}" ] && {
    echo "Usage: ${0} <esp-namespace> <grafana-namespace>" >&2
    exit 1
}

#Work out the domain names
. get-domain-name.sh $ESP_NAMESPACE $GRAFANA_NAMESPACE

function remove_oauth_client () {
  #TODO work out how to remove grafana form saslogon is it a dleete request
  echo "Not implemented"
}

cat <<EOF
OAuth details:
  ESP Domain:         ${ESP_DOMAIN}
  Grafana Domain:      ${GRAFANA_DOMAIN}
  OAuth client ID:     ${OAUTH_CLIENT_ID}
  OAuth client secret: ${OAUTH_CLIENT_SECRET}
EOF

remove_oauth_client
