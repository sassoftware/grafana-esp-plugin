ESP_NAMESPACE=""
GRAFANA_NAMESPACE=""
OAUTH_TYPE="viya"
CONTOUR_PROXY="false"
KEYCLOAK_SUBPATH="keycloak"
UNINSTALL_GRAFANA="false"
DRY_RUN="false"
KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"
ENABLE_NODE_SELECTOR="false"
ENABLE_DATASOURCES="false"
INSTALL_GRAFANA="true"

print_usage() {
    echo "Usage: $0 -n <esp-namespace> [options]" >&2
    echo "Default behavior: installs Grafana. Use -u/--uninstall-grafana to uninstall." >&2
    echo "Options:" >&2
    echo "  -g, --grafana-namespace <name>" >&2
    echo "  -o, --oauth-type <viya|keycloak>" >&2
    echo "  -c, --contour-proxy" >&2
    echo "  -k, --keycloak-subpath <path>" >&2
    echo "  -u, --uninstall-grafana" >&2
    echo "  -f, --kubeconfig <path>" >&2
    echo "  -d, --dry-run" >&2
    echo "  -s, --enable-node-selector" >&2
    echo "  -e, --enable-datasources" >&2
}

require_arg() {
    local flag="$1"
    local value="${2-}"
    if [[ -z "$value" || "$value" == -* ]]; then
        echo "Option $flag requires an argument." >&2
        print_usage
        exit 1
    fi
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--esp-namespace)
            require_arg "$1" "${2-}"
            ESP_NAMESPACE="$2"
            shift 2
            ;;
        -g|--grafana-namespace)
            require_arg "$1" "${2-}"
            GRAFANA_NAMESPACE="$2"
            shift 2
            ;;
        -o|--oauth-type)
            require_arg "$1" "${2-}"
            OAUTH_TYPE="$2"
            shift 2
            ;;
        -k|--keycloak-subpath)
            require_arg "$1" "${2-}"
            KEYCLOAK_SUBPATH="$2"
            shift 2
            ;;
        -f|--kubeconfig)
            require_arg "$1" "${2-}"
            KUBECONFIG="$2"
            shift 2
            ;;
        -c|--contour-proxy)
            CONTOUR_PROXY="true"
            shift
            ;;
        -u|--uninstall-grafana)
            UNINSTALL_GRAFANA="true"
            shift
            ;;
        -d|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -s|--enable-node-selector)
            ENABLE_NODE_SELECTOR="true"
            shift
            ;;
        -e|--enable-datasources)
            ENABLE_DATASOURCES="true"
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            print_usage
            exit 1
            ;;
    esac
done

if [ -z "$ESP_NAMESPACE" ]; then
    print_usage
    exit 1
fi

export ESP_NAMESPACE
export GRAFANA_NAMESPACE
export OAUTH_TYPE
export CONTOUR_PROXY
export KEYCLOAK_SUBPATH
export UNINSTALL_GRAFANA
export DRY_RUN
export ENABLE_NODE_SELECTOR
export ENABLE_DATASOURCES
export KUBECONFIG
export INSTALL_GRAFANA

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
    
    if [ "$DRY_RUN" == "false" ]; then
        
        if [ "$OAUTH_TYPE" == "viya" ]; then
            source ./register-oauth-client-viya.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE"
        else
            source ./register-oauth-client-keycloak.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE"
        fi

    fi
    
    bash configure-grafana.sh "$ESP_NAMESPACE" "$GRAFANA_NAMESPACE" "$GRAFANA_PLUGIN_VERSION"
fi