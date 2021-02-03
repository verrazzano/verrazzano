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
    echo " -z tar_gz_file   Name of the compressed tar file to generate. Ie: capture.tar.gz"
    echo " -h               Help"
    echo ""
    exit 1
}

kubectl >/dev/null 2>&1 || {
  echo "kubectl is required but cannot be found on the path. Aborting."
  exit 1
}

TAR_GZ_FILE=""
DUMP_SECRETS="FALSE"
while getopts z:sh flag
do
    case "${flag}" in
        z) TAR_GZ_FILE=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done
shift $((OPTIND -1))

if [ -z "$TAR_GZ_FILE" ] ; then
  usage
fi

if [ -f "$TAR_GZ_FILE" ] ; then
  echo "$TAR_GZ_FILE already exists. Aborting."
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
#	application-configurations.json
#	crd.json
#	gateways.json
#	helm-ls.json
#	helm-version.out
#	ingress.json
#	ingress-traits.json
#	nodes.json
#	pv.json
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

CAPTURE_DIR=$(mktemp -d $(pwd)/capture_XXXXXXX)
if [ -z $CAPTURE_DIR ] || [ ! -d $CAPTURE_DIR ]; then
  echo "Failed to intialize temporary directory"
  exit 1
fi

function full_k8s_cluster_dump() {
  echo "Full capture of kubernetes cluster"
  # Get general cluster-info dump, this contains quite a bit but not everything, it also sets up the directory structure
  kubectl cluster-info dump --all-namespaces --output-directory=$CAPTURE_DIR/cluster-dump >/dev/null 2>&1
  if [ $? -eq 0 ]; then
    kubectl version -o json > $CAPTURE_DIR/cluster-dump/kubectl-version.json || true
    kubectl get crd -o json > $CAPTURE_DIR/cluster-dump/crd.json || true
    kubectl get pv -o json > $CAPTURE_DIR/cluster-dump/pv.json || true
    kubectl get ingress -A -o json > $CAPTURE_DIR/cluster-dump/ingress.json || true
    kubectl get ApplicationConfiguration --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/application-configurations.json || true
    kubectl get IngressTrait --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/ingress-traits.json || true
    kubectl get Coherence --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/coherence.json || true
    kubectl get gateway --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/gateways.json || true
    kubectl get virtualservice --all-namespaces -o json > $CAPTURE_DIR/cluster-dump/virtualservices.json || true
    kubectl describe verrazzano --all-namespaces > $CAPTURE_DIR/cluster-dump/verrazzano_resources.out || true
    kubectl api-resources -o wide > $CAPTURE_DIR/cluster-dump/api_resources.out || true
    kubectl describe configmap --all-namespaces > $CAPTURE_DIR/cluster-dump/configmaps.out || true
    helm version > $CAPTURE_DIR/cluster-dump/helm-version.out || true
    helm ls -A -o json > $CAPTURE_DIR/cluster-dump/helm-ls.json || true
  else
    echo "Failed to dump cluster, verify kubectl has access to the cluster"
  fi
}

function save_dump_file() {
  # We only save files into cluster-dump and below we do not save the temp directory portion
  if [ -d $CAPTURE_DIR/cluster-dump ]; then
    tar -czf $TAR_GZ_FILE -C $CAPTURE_DIR cluster-dump
    echo "Dump saved to $TAR_GZ_FILE"
  fi
}

function cleanup_dump() {
  rm -rf $CAPTURE_DIR
}

full_k8s_cluster_dump
if [ $? -eq 0 ]; then
  save_dump_file
fi

cleanup_dump
