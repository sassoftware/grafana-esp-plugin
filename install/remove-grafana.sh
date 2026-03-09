#!/usr/bin/env bash

set -e -o pipefail -o nounset

NAMESPACE="${1}"

[ -z "${KUBECONFIG-}" ] && {
  echo "KUBECONFIG environment variable unset." >&2
  exit 1
}

[ -d "./manifests" ] || {
  echo "No manifest directory found." >&2
  exit 1
}

[ -z "${NAMESPACE-}" ] && {
  echo "Usage: ${0} <namespace> <version>" >&2
  exit 1
}

echo "Removing Grafana..."
kubectl -n "${NAMESPACE}" delete -f ./manifests/grafana.yaml || {
  echo ""
}

echo "Removing config map..."
kubectl -n "${NAMESPACE}" delete -f ./manifests/config-map.yaml >&2 || {
  echo ""
}

if [[ "${CONTOUR_PROXY}" ]]; then
  kubectl -n "${NAMESPACE}" delete -f ./manifests/grafana-http-proxy.yaml >&2 || {
    echo ""
  }
else
  kubectl -n "${NAMESPACE}" delete -f ./manifests/grafana-ingress.yaml >&2 || {
    echo ""
  }
fi
