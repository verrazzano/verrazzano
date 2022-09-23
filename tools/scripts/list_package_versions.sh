#!/bin/bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# tools/scripts/list_package_versions.sh list|table [<github-usernsame>:<github-token>]
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
TMP_DIR=$(mktemp -d)
TAGS_SIZE=100
IMAGE_LIST=$TMP_DIR/image_list.txt
$SCRIPT_DIR/generate_image_list.sh $SCRIPT_DIR/../../platform-operator/verrazzano-bom.json $IMAGE_LIST oracle,verrazzano
FORMAT="${1:-list}"
GH_CRED=
if [ -z "$2" ]; then
    # echo "Specify <github-usernsame>:<github-token> to get a higher github API rate limit"
    GH_CRED=
else
    GH_CRED="-u $2"
fi

function find_version_in_use() {
    image_name=$1
    if [ -z "$3" ]; then
        image_name=$1
    else
        image_name=$3
    fi
    cat $IMAGE_LIST | grep -m1 $image_name: | cut -d: -f2 | cut -d- -f1
}

# available_versions <component> <owner> [imageInBOM-component] [tags|releases]
function available_versions() {
    TAGS_FILE="$TMP_DIR/tags_$1.json"
    RES="${4:-tags}"
    if [ ! -f "$TAGS_FILE" ]; then
        curl -L -s $GH_CRED -H "Accept: application/json" https://api.github.com/repos/$2/$1/$RES?per_page=$TAGS_SIZE > $TAGS_FILE
    fi
    local versions
    if [ -z "$4" ]; then
      cat $TAGS_FILE | jq -r '.[] | .name'
    else
      cat $TAGS_FILE | jq -r '.[] | .tag_name'
    fi
}

function is_valid_verion() {
    # skip v010(no'.'), rc, beta, alpha, helm
    [[ "$1" == *"."* ]] && [[ "$1" != *"rc"* ]] && [[ "$1" != *"beta"* ]] && [[ "$1" != *"alpha"* ]] && [[ "$1" != *"helm"* ]]
}

function trim_version() {
    if [[ "$1" == *"-"* ]]; then
        local trimed=${1#*-}   # remove prefix ending in "-"
        trim_version $trimed
    else
        echo $1
    fi
}

# list_versions <component> <owner> [imageInBOM-component] [tags|releases]
function list_versions() {
    in_use=$(find_version_in_use $1 $2 $3)
    local versions=($(available_versions $1 $2 $3 $4))
    echo "$1 in use: $in_use"
    local found=false
    for ver in "${versions[@]}"; do
        if $(is_valid_verion $ver) && [[ "$found" == false ]]; then
            ver=$(trim_version $ver)
            if [[ "$ver" == *"$in_use"* ]]; then
                found=true
                ver="$ver (in use)"
            fi
            echo "    $ver"
        fi
    done
}

function table_row() {
    local in_use=$(find_version_in_use $1 $2 $3)
    local versions=($(available_versions $1 $2 $3 $4))
    local found=false
    local latest=""
    for ver in "${versions[@]}"; do
        if $(is_valid_verion $ver) && [[ "$found" == false ]]; then
            ver=$(trim_version $ver)
            if [[ "$latest" == "" ]]; then
                latest="$ver"
            fi
            if [[ "$ver" == *"$in_use"* ]]; then
                in_use="$ver"
            fi
        fi
    done
    RES="${4:-tags}"
    echo "${1},${in_use},${latest},https://github.com/$2/$1/$RES"
}

if [ "$FORMAT" == table ]; then
    echo "Component,In Use,Latest, Notes"
fi

function package_versions() {
    if [ "$FORMAT" == list ]; then
        list_versions $1 $2 $3 $4
    elif [[ "$FORMAT" == table ]]; then
        table_row $1 $2 $3 $4
    else
        echo "Unknown format: $FORMAT"
    fi
}

package_versions alertmanager prometheus alertmanager
package_versions backup-restore-operator rancher rancher-backup
package_versions cert-manager cert-manager cert-manager-controller
package_versions coherence-operator oracle
package_versions external-dns kubernetes-sigs
package_versions fluentd fluent fluentd-kubernetes-daemonset
package_versions grafana grafana
package_versions ingress-nginx kubernetes nginx-ingress-controller releases
package_versions istio istio proxyv2
package_versions jaeger jaegertracing jaeger
package_versions jaeger jaegertracing jaeger-operator
package_versions keycloak keycloak
package_versions kiali kiali
package_versions kube-state-metrics kubernetes
package_versions mysql-server mysql mysql
package_versions node_exporter prometheus node-exporter
package_versions oam-kubernetes-runtime crossplane
package_versions OpenSearch opensearch-project opensearch
package_versions OpenSearch-Dashboards opensearch-project opensearch-dashboards
package_versions prometheus prometheus
package_versions prometheus-adapter kubernetes-sigs
package_versions prometheus-operator prometheus-operator
package_versions pushgateway prometheus pushgateway
package_versions rancher rancher
package_versions velero vmware-tanzu velero
package_versions velero-plugin-for-aws vmware-tanzu
package_versions weblogic-kubernetes-operator oracle

