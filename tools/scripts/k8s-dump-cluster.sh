#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

# REVIEW: Look at whether we can use the common.sh utility functions here (there is some log support but
# that seems intertwined with the build/install, not sure it is a good fit here as this is intended to be
# standalone capture as well not specific).

# prints usage message for this script to consoleerr
# Usage:
# usage
function usage {
    echo ""
    echo "usage: $0 -z tar_gz_file"
    echo " You must specify at least a tar file or a directory to capture into"
    echo " Specifying both -z and -d is valid as well, but note they are independent of each other"
    echo " -z tar_gz_file   Name of the compressed tar file to generate. Ie: capture.tar.gz"
    echo " -d directory     Directory to capture an expanded dump into. This does not affect a tar_gz_file if that is also specified"
    echo " -a               Call the analyzer on the captured dump and report to stdout"
    echo " -r report_file   Call the analyzer on the captured dump and report to the file specified"
    echo " -h               Help"
    echo ""
    exit 1
}

kubectl >/dev/null 2>&1 || {
  echo "kubectl is required but cannot be found on the path. Aborting."
  exit 1
}

TAR_GZ_FILE=""
ANALYZE="FALSE"
REPORT_FILE=""
while getopts z:d:har: flag
do
    case "${flag}" in
        z) TAR_GZ_FILE=${OPTARG};;
        d) DIRECTORY=${OPTARG};;
        a) ANALYZE="TRUE";;
        r) REPORT_FILE=${OPTARG}
           ANALYZE="TRUE";;
        h) usage;;
        *) usage;;
    esac
done
shift $((OPTIND -1))

# We need at least a directory or a tar file specified for the dump
if [[ -z "$TAR_GZ_FILE" && -z "$DIRECTORY" ]] ; then
  usage
fi

# If a tar file output was specified and it exists already fail
if [[ ! -z "$TAR_GZ_FILE" && -f "$TAR_GZ_FILE" ]] ; then
  echo "$TAR_GZ_FILE already exists. Aborting."
  exit 1
fi

# If a tar file output was specified and it exists already fail
if [[ ! -z "$DIRECTORY" && -f "$DIRECTORY" ]] ; then
  echo "$DIRECTORY already exists. Aborting."
  exit 1
fi

# If a report file output was specified and it exists already fail
if [[ ! -z "$REPORT_FILE" && -f "$REPORT_FILE" ]] ; then
  echo "$REPORT_FILE already exists. Aborting."
  exit 1
fi

# We create a temporary directory to dump info. The basic structure is along these lines.
#
# $CAPTURE_DIR/cluster-dump
#	directory per namespace
#		daemonsets.json
#		deployments.json
#		events.json
#		pods.json
#		replicasets.json
#		replication-controllers.json
#		services.json
#		directory per pod
#			logs.txt
#       api-resources.out
#	application-configurations.json
#       coherence.json
#       configmaps.out
#	crd.json
#       es_indexex.out
#	gateways.json
#	helm-ls.json
#	helm-version.out
#       images-on-nodes.csv
#	ingress.json
#	ingress-traits.json
#       kubectl-version.json
#	nodes.json
#	pv.json
#       verrazzano_resources.out
#	virtualservices.json
#
# REVIEW: We certainly could capture some of the above per-namespace into the hierarchy
#         created by the cluster-info.
# NOTE: We are capturing details into json (a few version dumps aren't), this ultimately will be consumed by the triage
#       tooling but it is also human readable.
# EVOLVING: This is a first cut that captures everything (quick/easy), we may not want that to remain as an option
#      but by default we will really want to capture details about our namespaces, and capture some info otherwise.
#      So we will want to have some options to control what we capture here overall. Maybe:
#         base: This would be default and would capture Verrazzano related namespaces
#         full: This would
# REVIEW: As this is intended to be used to assist in issue handling, we do not want to capture things from a customer
#      environment which may be considered sensitive. The intention is that both the capture and triage tooling ultimately
#      would be runnable by the customer entirely (ie: we would never receive the captured data), but we need to be
#      careful in any case as once captured into an archive they need to be aware in how they handle it, and we may
#      need to trim down more from what we capture as well.

if [ -z $DIRECTORY ]; then
  CAPTURE_DIR=$(mktemp -d $(pwd)/capture_XXXXXXX)
else
  mkdir $DIRECTORY
  CAPTURE_DIR=$DIRECTORY
fi

if [ -z $CAPTURE_DIR ] || [ ! -d $CAPTURE_DIR ]; then
  echo "Failed to intialize capture directory"
  exit 1
fi

function process_nodes_output() {
  if [ -f $CAPTURE_DIR/cluster-dump/nodes.json ]; then
    cat $CAPTURE_DIR/cluster-dump/nodes.json | jq '.items[].status.images[].names|@csv' | sed -e 's/"//g' -e 's/\\//g'| sort -u > $CAPTURE_DIR/cluster-dump/images-on-nodes.csv
  fi
}

function dump_es_indexes() {
  kubectl --insecure-skip-tls-verify get ingress -A -o json | jq .items[].spec.tls[].hosts[]  2>/dev/null | grep elasticsearch.vmi.system.default | sed -e 's;^";https://;' -e 's/"//' || true
  local ES_ENDPOINT=$(kubectl --insecure-skip-tls-verify get ingress -A -o json | jq .items[].spec.tls[].hosts[] 2>/dev/null | grep elasticsearch.vmi.system.default | sed -e 's;^";https://;' -e 's/"//') || true
  local ES_USER=$(kubectl --insecure-skip-tls-verify get secret -n verrazzano-system verrazzano -o jsonpath={.data.username} 2>/dev/null | base64 --decode) || true
  local ES_PWD=$(kubectl --insecure-skip-tls-verify get secret -n verrazzano-system verrazzano -o jsonpath={.data.password} 2>/dev/null | base64 --decode) || true
  if [ ! -z $ES_ENDPOINT ] && [ ! -z $ES_USER ] && [ ! -z $ES_PWD ]; then
    curl -k -u $ES_USER:$ES_PWD $ES_ENDPOINT/_all || true
  fi
}

# This relies on the directory structure which is setup by kubectl cluster-info dump, so this is not a standalone function and currenntly
# should only be called after that has been called
function dump_configmaps() {
  # Get list of all config maps in the cluster
  kubectl --insecure-skip-tls-verify get -o custom-columns=NAMESPACEHEADER:.metadata.namespace,NAMEHEADER:.metadata.name configmap --all-namespaces > $CAPTURE_DIR/cluster-dump/configmap_list.out || true

  # Iterate the list, describe each configmap individually in a file in the namespace
  local CSV_LINE=""
  local NAMESPACE=""
  local CONFIGNAME=""
  while read INPUT_LINE; do
      if [[ ! $INPUT_LINE == *"NAMESPACEHEADER"* ]]; then
        CSV_LINE=$(echo "$INPUT_LINE" | sed  -e "s/[' '][' ']*/,/g")
        NAMESPACE=$(echo "$CSV_LINE" | cut -d, -f"1")
        CONFIGNAME=$(echo "$CSV_LINE" | cut -d, -f"2")
        if [ ! -z $NAMESPACE ] && [ ! -z $CONFIGNAME ] ; then
          kubectl --insecure-skip-tls-verify describe configmap $CONFIGNAME -n $NAMESPACE > $CAPTURE_DIR/cluster-dump/$NAMESPACE/$CONFIGNAME.configmap || true
        fi
      fi
    done <$CAPTURE_DIR/cluster-dump/configmap_list.out
}

# This relies on the directory structure which is setup by kubectl cluster-info dump, so this is not a standalone function and currently
# should only be called after that has been called.
# kubectl cluster-info dump only captures certain information, we need additional information captured though and have it placed into
# namespace specific directories which are created by cluster-info. We capture those things here.
#
function dump_extra_details_per_namespace() {
  # Get list of all namespaces in the cluster
  kubectl --insecure-skip-tls-verify get -o custom-columns=NAMEHEADER:.metadata.name namespaces > $CAPTURE_DIR/cluster-dump/namespace_list.out || true

  # Iterate the list, describe each configmap individually in a file in the namespace
  local NAMESPACE=""
  while read NAMESPACE; do
    if [[ ! $NAMESPACE == *"NAMEHEADER"* ]]; then
      if [ ! -z $NAMESPACE ] ; then
        echo "Capturing $NAMESPACE namespace"
        # The cluster-dump should create the directories for us, but just in case there is a situation where there is a namespace
        # that is present which doesn't have one created, make sure we have the directory
        if [ ! -d $CAPTURE_DIR/cluster-dump/$NAMESPACE ] ; then
          mkdir $CAPTURE_DIR/cluster-dump/$NAMESPACE || true
        fi
        kubectl --insecure-skip-tls-verify get ApplicationConfiguration -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/application-configurations.json || true
        kubectl --insecure-skip-tls-verify get IngressTrait -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/ingress-traits.json || true
        kubectl --insecure-skip-tls-verify get Coherence -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/coherence.json || true
        kubectl --insecure-skip-tls-verify get gateway -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/gateways.json || true
        kubectl --insecure-skip-tls-verify get virtualservice -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/virtualservices.json || true
        kubectl --insecure-skip-tls-verify describe verrazzano -n $NAMESPACE 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/verrazzano_resources.json || true
        kubectl --insecure-skip-tls-verify get rolebindings -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/role-bindings.json || true
        kubectl --insecure-skip-tls-verify get clusterrolebindings -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/cluster-role-bindings.json || true
        kubectl --insecure-skip-tls-verify get clusterroles -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/cluster-roles.json || true
        kubectl --insecure-skip-tls-verify get ns -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/namespace.json || true
        kubectl --insecure-skip-tls-verify get multiclusterapplicationconfigurations -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/multicluster-application-configurations.json || true
        kubectl --insecure-skip-tls-verify get multiclustercomponents -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/multicluster-components.json || true
        kubectl --insecure-skip-tls-verify get multiclusterconfigmaps -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/multicluster-config-maps.json || true
        kubectl --insecure-skip-tls-verify get multiclusterloggingscopes -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/multicluster-logging-scopes.json || true
        kubectl --insecure-skip-tls-verify get multiclustersecrets -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/multicluster-secrets.json || true
        kubectl --insecure-skip-tls-verify get clusterroles -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/$NAMESPACE/cluster-roles.json || true
      fi
    fi
  done <$CAPTURE_DIR/cluster-dump/namespace_list.out
  rm $CAPTURE_DIR/cluster-dump/namespace_list.out
}

function full_k8s_cluster_dump() {
  echo "Full capture of kubernetes cluster"
  # Get general cluster-info dump, this contains quite a bit but not everything, it also sets up the directory structure
  kubectl --insecure-skip-tls-verify cluster-info dump --all-namespaces --output-directory=$CAPTURE_DIR/cluster-dump >/dev/null 2>&1
  if [ $? -eq 0 ]; then
    kubectl --insecure-skip-tls-verify version -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/kubectl-version.json || true
    kubectl --insecure-skip-tls-verify get crd -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/crd.json || true
    kubectl --insecure-skip-tls-verify get pv -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/pv.json || true
    kubectl --insecure-skip-tls-verify get ingress -A -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/ingress.json || true
    kubectl --insecure-skip-tls-verify api-resources -o wide 2>/dev/null > $CAPTURE_DIR/cluster-dump/api_resources.out || true
    kubectl --insecure-skip-tls-verify get netpol -A -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/network-policies.json || true
    kubectl --insecure-skip-tls-verify describe netpol -A 2>/dev/null > $CAPTURE_DIR/cluster-dump/network-policies.txt || true
    # squelch the "too many clients" warnings from newer kubectl versions
    dump_extra_details_per_namespace
    dump_configmaps
    helm version 2>/dev/null > $CAPTURE_DIR/cluster-dump/helm-version.out || true
    helm ls -A -o json 2>/dev/null > $CAPTURE_DIR/cluster-dump/helm-ls.json || true
    dump_es_indexes > $CAPTURE_DIR/cluster-dump/es_indexes.out || true
    process_nodes_output || true
  else
    echo "Failed to dump cluster, verify kubectl has access to the cluster"
  fi
}

function analyze_dump() {
  if [ $ANALYZE == "TRUE" ]; then
    if ! [ -x "$(command -v go)" ]; then
      echo "Analyze requires go which does not appear to be installed, skipping analyze"
    else
      local FULL_PATH_CAPTURE_DIR=$(echo "$(cd "$(dirname "$CAPTURE_DIR")" && pwd -P)/$(basename "$CAPTURE_DIR")")
      local SAVE_DIR=$(pwd)
      cd $SCRIPT_DIR/../analysis
      # To enable debug, add  -zap-log-level debug
      if [ -z $REPORT_FILE ]; then
        GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go --analysis=cluster --info=true $FULL_PATH_CAPTURE_DIR || true
      else
        # Since we have to change the current working directory to run go, we need to take into account if the reportFile specified was relative to the original
        # working directory. If it was absolute then we just use it directly
        if [[ $REPORT_FILE = /* ]]; then
          GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go --analysis=cluster --info=true --reportFile=$REPORT_FILE $FULL_PATH_CAPTURE_DIR || true
        else
          GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go --analysis=cluster --info=true --reportFile=$SAVE_DIR/$REPORT_FILE $FULL_PATH_CAPTURE_DIR || true
        fi
      fi
      cd $SAVE_DIR
    fi
  fi
}

function save_dump_file() {
  # This will save the dump to a tar gz file if that was specified
  if [ ! -z $TAR_GZ_FILE ]; then
    # We only save files into cluster-dump and below we do not save the temp directory portion
    if [ -d $CAPTURE_DIR/cluster-dump ]; then
      tar -czf $TAR_GZ_FILE -C $CAPTURE_DIR cluster-dump
      echo "Dump saved to $TAR_GZ_FILE"
    fi
  fi
}

function cleanup_dump() {
  # This will cleanup the capture directory if it was not specified (it is a temp directory in that case)
  if [ -z $DIRECTORY ]; then
    rm -rf $CAPTURE_DIR
  fi
}

full_k8s_cluster_dump
if [ $? -eq 0 ]; then
  save_dump_file
fi

analyze_dump
cleanup_dump
