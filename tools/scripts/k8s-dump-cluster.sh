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
        r) REPORT_FILE=${OPTARG};;
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
  local ES_USER=$(kubectl --insecure-skip-tls-verify get secret -n verrazzano-system verrazzano -o jsonpath={.data.username} | base64 --decode) || true
  local ES_PWD=$(kubectl --insecure-skip-tls-verify get secret -n verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode) || true
  if [ ! -z $ES_ENDPOINT ] && [ ! -z $ES_USER ] && [ ! -z $ES_PWD ]; then
    curl -k -u $ES_USER:$ES_PWD $ES_ENDPOINT/_all
  fi
}

function full_k8s_cluster_dump() {
  echo "Full capture of kubernetes cluster"
  # Get general cluster-info dump, this contains quite a bit but not everything, it also sets up the directory structure
  kubectl --insecure-skip-tls-verify cluster-info dump --all-namespaces --output-directory=$CAPTURE_DIR/cluster-dump >/dev/null 2>&1
  if [ $? -eq 0 ]; then
    kubectl --insecure-skip-tls-verify version -o json > $CAPTURE_DIR/cluster-dump/kubectl-version.json || true
    kubectl --insecure-skip-tls-verify get crd -o json > $CAPTURE_DIR/cluster-dump/crd.json || true
    kubectl --insecure-skip-tls-verify get pv -o json > $CAPTURE_DIR/cluster-dump/pv.json || true
    kubectl --insecure-skip-tls-verify get ingress -A -o json > $CAPTURE_DIR/cluster-dump/ingress.json || true
    kubectl --insecure-skip-tls-verify get ApplicationConfiguration --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/application-configurations.json || true
    kubectl --insecure-skip-tls-verify get IngressTrait --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/ingress-traits.json || true
    kubectl --insecure-skip-tls-verify get Coherence --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/coherence.json || true
    kubectl --insecure-skip-tls-verify get gateway --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/gateways.json || true
    kubectl --insecure-skip-tls-verify get virtualservice --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/virtualservices.json || true
    kubectl --insecure-skip-tls-verify describe verrazzano --all-namespaces > $CAPTURE_DIR/cluster-dump/verrazzano_resources.json || true
    kubectl --insecure-skip-tls-verify api-resources -o wide > $CAPTURE_DIR/cluster-dump/api_resources.out || true
    kubectl --insecure-skip-tls-verify get rolebindings --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/role-bindings.json || true
    kubectl --insecure-skip-tls-verify get clusterrolebindings --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/cluster-role-bindings.json || true
    kubectl --insecure-skip-tls-verify get clusterroles --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/cluster-roles.json || true
    # squelch the "too many clients" warnings from newer kubectl versions
    kubectl --insecure-skip-tls-verify describe configmap --all-namespaces > $CAPTURE_DIR/cluster-dump/configmaps.out 2> /dev/null || true
    helm version > $CAPTURE_DIR/cluster-dump/helm-version.out || true
    helm ls -A -o json > $CAPTURE_DIR/cluster-dump/helm-ls.json || true
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
        echo "DEBUG1 $REPORT_FILE      $FULL_CAPTURE_PATH_DIR"
        GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go --analysis=cluster --info=true $FULL_PATH_CAPTURE_DIR || true
      else
        echo "DEBUG2 $REPORT_FILE      $FULL_CAPTURE_PATH_DIR"
        GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go --analysis=cluster --info=true --reportFile=$REPORT_FILE $FULL_PATH_CAPTURE_DIR || true
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
