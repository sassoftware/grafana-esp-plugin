#!/usr/bin/env bash

set -e -o pipefail -o nounset

[ -z "${KUBECONFIG-}" ] && {
    echo "KUBECONFIG environment variable unset." >&2
    exit 1
}

[ -d "./manifests" ] || {
    echo "No manifest directory found." >&2
    exit 1
}

echo "Removing Grafana..."
kubectl -n "${ESP_NAMESPACE}" delete -k ./manifests/
