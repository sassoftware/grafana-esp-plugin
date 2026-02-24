#!/usr/bin/env bash

set -e -o pipefail -o nounset
set -o nounset

echo "Determining domain names"

#input variables
ESP_NAMESPACE="${1}"
GRAFANA_NAMESPACE="${2:-${ESP_NAMESPACE}}"

# If no esp domain then we are looking to install grafana on a separate namespace
if [ -z ${ESP_DOMAIN+null} ]; then
  # We cant easily determine the grafana domain unless there is an ingress

  ESP_DOMAIN=$(kubectl -n "${ESP_NAMESPACE}" get ingress/sas-event-stream-manager-app --output json | jq -r '.spec.rules[0].host') || {
    echo "Failed to get ESP domain from ingress, trying http proxy"
  }

  if [ -z "${ESP_DOMAIN}" ]; then
    ESP_DOMAIN=$(kubectl get httpproxy -n "${ESP_NAMESPACE}" sas-httpproxy-root -o jsonpath="{.spec.virtualhost.fqdn}") || {
      echo "Failed to get ESP domain from http proxy"
    }
  fi

  if [ "${ESP_DOMAIN}" == null ]; then
    echo "Unable to determine the esp domain name from an ingress, please set ESP_DOMAIN to your environments domain name." >&2
    exit 1
  fi

  echo "Domain is " $ESP_DOMAIN
fi

if [ "$ESP_NAMESPACE" == "$GRAFANA_NAMESPACE" ]; then
  GRAFANA_DOMAIN=$ESP_DOMAIN
fi

# If no grafana domain then we are looking to install grafana on a separate namespace
[ -z ${GRAFANA_DOMAIN+null} ] && {

  # We cant easily determine the grafana domain unless there is an ingress
  GRAFANA_DOMAIN=$(kubectl -n "${GRAFANA_NAMESPACE}" get ingress/sas-event-stream-manager-app --output json | jq -r '.items[0].spec.rules[0].host') || {
    echo "Failed to get ESP domain from ingress trying http proxy" >&2
    GRAFANA_DOMAIN=$(kubectl get httpproxy -n "${GRAFANA_NAMESPACE}" sas-httpproxy-root -o jsonpath="{.spec.virtualhost.fqdn}")
    echo "Domain is " $GRAFANA_DOMAIN
  }

  if [ "${GRAFANA_DOMAIN}" == null ]; then
    echo "Unable to determine the grafana domain name from an ingress, please set GRAFANA_DOMAIN to your environments domain name." >&2
    exit 1
  fi
}

export ESP_DOMAIN
export GRAFANA_DOMAIN
