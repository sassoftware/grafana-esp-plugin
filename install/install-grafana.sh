ESP_NAMESPACE=""
GRAFANA_NAMESPACE=""
OAUTH_TYPE="viya"
CONTOUR_PROXY="false"
KEYCLOAK_SUBPATH="keycloak"
INSTALL_GRAFANA="true"
UNINSTALL_GRAFANA="false"
KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"

# Short options:
# -n <esp-namespace>
# -g <grafana-namespace>
# -o <oauth-type>
# -c <contour-proxy>
# -k <keycloak-subpath>
# -i <install-grafana>
# -u <uninstall-grafana>
# -f <kubeconfig-file>
while getopts ":n:g:o:c:k:i:u:f:" opt; do
    case "$opt" in
        n) ESP_NAMESPACE="$OPTARG" ;;
        g) GRAFANA_NAMESPACE="$OPTARG" ;;
        o) OAUTH_TYPE="$OPTARG" ;;
        c) CONTOUR_PROXY="$OPTARG" ;;
        k) KEYCLOAK_SUBPATH="$OPTARG" ;;
        i) INSTALL_GRAFANA="$OPTARG" ;;
        u) UNINSTALL_GRAFANA="$OPTARG" ;;
        f) KUBECONFIG="$OPTARG" ;;
        \?) echo "Unknown option: -$OPTARG" >&2; exit 1 ;;
        :) echo "Option -$OPTARG requires an argument." >&2; exit 1 ;;
    esac
done
shift $((OPTIND - 1))

# Backward-compatible positional arguments
ESP_NAMESPACE=${ESP_NAMESPACE:-${1}}
GRAFANA_NAMESPACE=${GRAFANA_NAMESPACE:-${2:-${ESP_NAMESPACE}}}
OAUTH_TYPE=${OAUTH_TYPE:-${3:-viya}}
CONTOUR_PROXY=${CONTOUR_PROXY:-${4:-false}}
KEYCLOAK_SUBPATH=${KEYCLOAK_SUBPATH:-${5:-keycloak}}
INSTALL_GRAFANA=${INSTALL_GRAFANA:-${6:-true}}
UNINSTALL_GRAFANA=${UNINSTALL_GRAFANA:-${7:-false}}

if [ -z "$ESP_NAMESPACE" ]; then
    echo "Usage: $0 -n <esp-namespace> [-g <grafana-namespace>] [-o <oauth-type:viya|keycloak>] [-c <contour-proxy:boolean>] [-k <keycloak-subpath>] [-i <install-grafana:boolean>] [-u <uninstall-grafana:boolean>]" >&2
    exit 1
fi

export ESP_NAMESPACE
export GRAFANA_NAMESPACE
export OAUTH_TYPE
export CONTOUR_PROXY
export KEYCLOAK_SUBPATH
export INSTALL_GRAFANA
export UNINSTALL_GRAFANA

# get latest grafana plugin version
LATEST_RELEASE=`curl -X GET -s -k https://api.github.com/repos/sassoftware/grafana-esp-plugin/releases | jq -r 'first | .tag_name'`

if [ -z "$GRAFANA_PLUGIN_VERSION" ]; then
    GRAFANA_PLUGIN_VERSION=$LATEST_RELEASE
fi

if [ "$UNINSTALL_GRAFANA" == "true" ]; then
    
    if [ "$OAUTH_TYPE" == "viya" ]; then
        bash remove-oauth-viya.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE"
    else
        bash remove-oauth-keycloak.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE"
    fi
    
    export DRY_RUN=true
    bash configure-grafana.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE" "$GRAFANA_PLUGIN_VERSION"
    bash remove-grafana.sh "$GRAFANA_NAMESPACE"
else
    
    if [ "$OAUTH_TYPE" == "viya" ]; then
        source ./register-oauth-client-viya.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE"
    else
        source ./register-oauth-client-keycloak.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE"
    fi
    
    bash configure-grafana.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE" "$GRAFANA_PLUGIN_VERSION"
fi