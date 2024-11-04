#!/usr/bin/env bash

set -e -o pipefail -o nounset

#input variables
ESP_NAMESPACE="${1}"; export ESP_NAMESPACE
GRAFANA_NAMESPACE="${2:-${ESP_NAMESPACE}}"
ESP_PLUGIN_VERSION="${3}"

#optional environment variables - exported for use in other scripts
OAUTH_CLIENT_ID="${OAUTH_CLIENT_ID:-sv_client}"; export OAUTH_CLIENT_ID
OAUTH_CLIENT_SECRET="${OAUTH_CLIENT_SECRET:-secret}"; export OAUTH_CLIENT_SECRET
KEYCLOAK_SUBPATH="${KEYCLOAK_SUBPATH:-auth}"; export KEYCLOAK_SUBPATH

#optional environment variables
OAUTH_TYPE="${OAUTH_TYPE:-viya}";
DRY_RUN="${DRY_RUN:-false}"
INSTALL_GRAFANA="${INSTALL_GRAFANA:-false}"
GRAFANA_VERSION="${GRAFANA_VERSION:-11.3.0}"

function check_requirements() {
  [ -z "${KUBECONFIG-}" ] && {
      echo "KUBECONFIG environment variable unset." >&2
      exit 1
  }

  [ -z "${ESP_NAMESPACE-}" ] && {
      echo "Usage: ${0} <esp-namespace> <grafana-namespace> <version>" >&2
      exit 1
  }

  [ -z "${ESP_PLUGIN_VERSION-}" ] && {
      echo "Usage: ${0} <esp-namespace> <grafana-namespace> <version>" >&2
      exit 1
  }

  if ! kubectl get namespace "${ESP_NAMESPACE}" 2>/dev/null 1>&2; then
      echo >&2 "ERROR: Namespace ${ESP_NAMESPACE} not found."
      exit 1
  fi

  if ! kubectl get namespace "${GRAFANA_NAMESPACE}" 2>/dev/null 1>&2; then
        echo >&2 "ERROR: Namespace ${GRAFANA_NAMESPACE} not found."
        exit 1
  fi
}

function generate_manifests() {
  if [ -d "./manifests" ]; then
      echo "Existing manifest directory found." >&2
      echo "Removing manifests..."
      rm -r ./manifests/
  fi

  [ -d "./manifests" ] || mkdir "manifests"
  cp -r *.yaml manifests/

  for file in `find ./manifests/ -name "*.y*ml"` ; do

    sed -i 's|TEMPLATE_AUTH_URL|'$TEMPLATE_AUTH_URL'|g' $file
    sed -i 's|TEMPLATE_TOKEN_URL|'$TEMPLATE_TOKEN_URL'|g' $file
    sed -i 's|TEMPLATE_API_URL|'$TEMPLATE_API_URL'|g' $file
    sed -i 's|TEMPLATE_SIGNOUT_REDIRECT_URL|'$TEMPLATE_SIGNOUT_REDIRECT_URL'|g' $file

    sed -i 's|TEMPLATE_GRAFANA_DOMAIN|'$GRAFANA_DOMAIN'|g' $file
    sed -i 's|TEMPLATE_ESP_DOMAIN|'$ESP_DOMAIN'|g' $file
    sed -i 's|TEMPLATE_OAUTH_CLIENT_ID|'$OAUTH_CLIENT_ID'|g' $file
    sed -i 's|TEMPLATE_OAUTH_CLIENT_SECRET|'$OAUTH_CLIENT_SECRET'|g' $file
    sed -i 's|TEMPLATE_ESP_PLUGIN_SOURCE|'$ESP_PLUGIN_SOURCE'|g' $file
    sed -i 's|TEMPLATE_GRAFANA_VERSION|'$GRAFANA_VERSION'|g' $file

    if [[ "${DRY_RUN}" == true ]]; then

      if [[ "${INSTALL_GRAFANA}" == false && "${file}" == "./manifests/grafana.yaml" ]]; then
        echo ""
      else
        echo $file
        cat $file
      fi

    fi

  done
}

check_requirements

echo "Fetching required deployment information..."

#Work out the domain names
. get-domain-name.sh $ESP_NAMESPACE $GRAFANA_NAMESPACE

ESP_PLUGIN_SOURCE="https://github.com/sassoftware/grafana-esp-plugin/releases/download/v$ESP_PLUGIN_VERSION/sasesp-plugin-$ESP_PLUGIN_VERSION.zip"

if [ "${OAUTH_TYPE}" == "viya" ]; then

  TEMPLATE_AUTH_URL="https://${ESP_DOMAIN}/SASLogon/oauth/authorize"
  TEMPLATE_TOKEN_URL="https://${ESP_DOMAIN}/SASLogon/oauth/token"
  TEMPLATE_API_URL="https://${ESP_DOMAIN}/SASLogon/userinfo"
  TEMPLATE_SIGNOUT_REDIRECT_URL="https://${ESP_DOMAIN}/SASLogon/logout.do"

elif [ "${OAUTH_TYPE}" == "keycloak" ]; then

  TEMPLATE_AUTH_URL="https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/realms/sas-esp/protocol/openid-connect/auth"
  TEMPLATE_TOKEN_URL="https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/realms/sas-esp/protocol/openid-connect/token"
  TEMPLATE_API_URL="https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/realms/sas-esp/protocol/openid-connect/userinfo"
  TEMPLATE_SIGNOUT_REDIRECT_URL="https://${ESP_DOMAIN}/${KEYCLOAK_SUBPATH}/realms/sas-esp/protocol/openid-connect/logout?client_id=${OAUTH_CLIENT_ID}\&post_logout_redirect_uri=https://${ESP_DOMAIN}/grafana/login"

else

  TEMPLATE_AUTH_URL="https://${ESP_DOMAIN}/uaa/oauth/authorize"
  TEMPLATE_TOKEN_URL="https://${ESP_DOMAIN}/uaa/oauth/token?token_format=jwt"
  TEMPLATE_API_URL="https://${ESP_DOMAIN}/uaa/userinfo"
  TEMPLATE_SIGNOUT_REDIRECT_URL="https://${ESP_DOMAIN}/oauth2/sign_out?rd=https://${ESP_DOMAIN}/uaa/logout.do?redirect=https://${ESP_DOMAIN}/uaa/login"

fi

cat <<EOF
Deployment details:
  ESP domain:          ${ESP_DOMAIN}
  Grafana domain:      ${GRAFANA_DOMAIN}
  OAuth client ID:     ${OAUTH_CLIENT_ID}
  OAuth client secret: ****
Deploying Grafana with values:
  ESP plugin source:   ${ESP_PLUGIN_SOURCE}
EOF

echo "Generating manifests..."
generate_manifests

if [[ "${DRY_RUN}" == true ]]; then
    exit 0
fi

echo "Create config-map.yaml"
kubectl -n "${GRAFANA_NAMESPACE}" apply -f ./manifests/config-map.yaml

if [[ "${INSTALL_GRAFANA}" == true ]]; then
  echo "Installing grafana"
  kubectl -n "${GRAFANA_NAMESPACE}" apply -f ./manifests/grafana.yaml
  #No need to patch grafana as it will already be installed with the plugin and config
  exit 0
fi

echo "Patching deployment/grafana with patch-grafana.yaml"
kubectl -n "${GRAFANA_NAMESPACE}" patch --patch-file ./manifests/patch-grafana.yaml deployment/grafana
