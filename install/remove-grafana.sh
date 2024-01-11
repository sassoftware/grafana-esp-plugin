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
kubectl -n "${NAMESPACE}" delete -k ./manifests/
