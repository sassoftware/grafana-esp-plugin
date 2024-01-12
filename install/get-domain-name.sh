#!/usr/bin/env bash

set -e -o pipefail -o nounset
set -o nounset

echo "Determining domain names"

#input variables
ESP_NAMESPACE="${1}";
GRAFANA_NAMESPACE="${2:-${ESP_NAMESPACE}}"

# If no esp domain then we are looking to install grafana on a separate namespace
if [ -z ${ESP_DOMAIN+null} ]; then
     # We cant easily determine the grafana domain unless there is an ingress
     ESP_DOMAIN=$(kubectl -n "${ESP_NAMESPACE}" get ingress/sas-event-stream-manager-app --output json | jq -r '.spec.rules[0].host')

     if [ "${ESP_DOMAIN}" == null ]; then
       echo "Unable to determine the esp domain name from an ingress, please set ESP_DOMAIN to your environments domain name." >&2
       exit 1
     fi
fi

if [ "$ESP_NAMESPACE" == "$GRAFANA_NAMESPACE" ]
then
    GRAFANA_DOMAIN=$ESP_DOMAIN
fi

# If no grafana domain then we are looking to install grafana on a separate namespace
[ -z ${GRAFANA_DOMAIN+null} ] && {

    # We cant easily determine the grafana domain unless there is an ingress
    GRAFANA_DOMAIN=$(kubectl -n "${GRAFANA_NAMESPACE}" get ingress --output json | jq -r '.items[0].spec.rules[0].host')

    if [ "${GRAFANA_DOMAIN}" == null ]; then
      echo "Unable to determine the grafana domain name from an ingress, please set GRAFANA_DOMAIN to your environments domain name." >&2
      exit 1
    fi
}

export ESP_DOMAIN
export GRAFANA_DOMAIN
