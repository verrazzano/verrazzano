#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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
    echo " -r report_file   Call the analyzer on the captured dump and report to the file specified, requires sources and go build environment"
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
# $CAPTURE_DIR/cluster-snapshot
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
#	  application-configurations.json
#   coherence.json
#	  gateways.json
#	  ingress-traits.json
#	  virtualservices.json
# configmap_list.out
#	crd.json
# es_indexes.out
# verrazzano-resources.json
#	helm-ls.json
#	helm-version.out
# images-on-nodes.csv
#	ingress.json
# kubectl-version.json
#	nodes.json
#	pv.json

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
  mkdir -p $DIRECTORY
  CAPTURE_DIR=$DIRECTORY
fi

if [ -z $CAPTURE_DIR ] || [ ! -d $CAPTURE_DIR ]; then
  echo "Failed to intialize capture directory"
  exit 1
fi

function process_nodes_output() {
  if [ -f $CAPTURE_DIR/cluster-snapshot/nodes.json ]; then
    cat $CAPTURE_DIR/cluster-snapshot/nodes.json | jq '.items[].status.images[].names|@csv' | sed -e 's/"//g' -e 's/\\//g'| sort -u > $CAPTURE_DIR/cluster-snapshot/images-on-nodes.csv
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
  kubectl --insecure-skip-tls-verify get -o custom-columns=NAMESPACEHEADER:.metadata.namespace,NAMEHEADER:.metadata.name configmap --all-namespaces > $CAPTURE_DIR/cluster-snapshot/configmap_list.out || true

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
          # The cluster-snapshot should create the directories for us, but just in case there is a situation where there is a namespace
          # that is present which doesn't have one created, make sure we have the directory
          if [ ! -d $CAPTURE_DIR/cluster-snapshot/$NAMESPACE ] ; then
            mkdir $CAPTURE_DIR/cluster-snapshot/$NAMESPACE || true
          fi
          kubectl --insecure-skip-tls-verify describe configmap $CONFIGNAME -n $NAMESPACE > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/$CONFIGNAME.configmap || true
        fi
      fi
    done <$CAPTURE_DIR/cluster-snapshot/configmap_list.out
}

# This relies on the directory structure which is setup by kubectl cluster-info dump, so this is not a standalone function and currently
# should only be called after that has been called.
# kubectl cluster-info dump only captures certain information, we need additional information captured though and have it placed into
# namespace specific directories which are created by cluster-info. We capture those things here.
#
function dump_extra_details_per_namespace() {
  # Get list of all namespaces in the cluster
  kubectl --insecure-skip-tls-verify get -o custom-columns=NAMEHEADER:.metadata.name namespaces > $CAPTURE_DIR/cluster-snapshot/namespace_list.out || true

  # Iterate the list, describe each configmap individually in a file in the namespace
  local NAMESPACE=""
  while read NAMESPACE; do
    if [[ ! $NAMESPACE == *"NAMEHEADER"* ]]; then
      if [ ! -z $NAMESPACE ] ; then
        echo "Capturing $NAMESPACE namespace"
        if ! kubectl get ns $NAMESPACE 2>&1 > /dev/null ; then
          echo "Namespace ${NAMESPACE} not found, skipping"
          continue
        fi
        # The cluster-snapshot should create the directories for us, but just in case there is a situation where there is a namespace
        # that is present which doesn't have one created, make sure we have the directory
        if [ ! -d $CAPTURE_DIR/cluster-snapshot/$NAMESPACE ] ; then
          mkdir $CAPTURE_DIR/cluster-snapshot/$NAMESPACE || true
        fi
        kubectl --insecure-skip-tls-verify get ApplicationConfiguration -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/application-configurations.json || true
        kubectl --insecure-skip-tls-verify get Component -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/components.json || true
        kubectl --insecure-skip-tls-verify get domains.weblogic.oracle -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/weblogic-domains.json || true
        kubectl --insecure-skip-tls-verify get IngressTrait -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/ingress-traits.json || true
        kubectl --insecure-skip-tls-verify get Coherence -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/coherence.json || true
        kubectl --insecure-skip-tls-verify get gateway -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/gateways.json || true
        kubectl --insecure-skip-tls-verify get virtualservice -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/virtualservices.json || true
        kubectl --insecure-skip-tls-verify get rolebindings -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/role-bindings.json || true
        kubectl --insecure-skip-tls-verify get clusterrolebindings -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/cluster-role-bindings.json || true
        kubectl --insecure-skip-tls-verify get clusterroles -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/cluster-roles.json || true
        kubectl --insecure-skip-tls-verify get ns $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/namespace.json || true
        kubectl --insecure-skip-tls-verify get pvc -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/persistent-volume-claims.json || true
        kubectl --insecure-skip-tls-verify get pv -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/persistent-volumes.json || true
        kubectl --insecure-skip-tls-verify get jobs.batch -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/jobs.json || true
        kubectl --insecure-skip-tls-verify get metricsbindings -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/metrics-bindings.json || true
        kubectl --insecure-skip-tls-verify get metricstemplates -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/metrics-templates.json || true
        kubectl --insecure-skip-tls-verify get multiclusterapplicationconfigurations -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/multicluster-application-configurations.json || true
        kubectl --insecure-skip-tls-verify get multiclustercomponents -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/multicluster-components.json || true
        kubectl --insecure-skip-tls-verify get multiclusterconfigmaps -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/multicluster-config-maps.json || true
        kubectl --insecure-skip-tls-verify get multiclusterloggingscopes -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/multicluster-logging-scopes.json || true
        kubectl --insecure-skip-tls-verify get multiclustersecrets -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/multicluster-secrets.json || true
        kubectl --insecure-skip-tls-verify get verrazzanoprojects -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/verrazzano-projects.json || true
        kubectl --insecure-skip-tls-verify get verrazzanomanagedclusters -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/verrazzano-managed-clusters.json || true
        kubectl --insecure-skip-tls-verify get verrazzanoweblogicworkload -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/verrazzano-weblogic-workload.json || true
        kubectl --insecure-skip-tls-verify get verrazzanocoherenceworkload -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/verrazzano-coherence-workload.json || true
        kubectl --insecure-skip-tls-verify get verrazzanohelidonworkload -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/verrazzano-helidon-workload.json || true
        kubectl --insecure-skip-tls-verify get domain -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/domain.json || true
        kubectl --insecure-skip-tls-verify get clusterroles -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/cluster-roles.json || true
        kubectl --insecure-skip-tls-verify get certificaterequests.cert-manager.io -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/certificate-requests.json || true
        kubectl --insecure-skip-tls-verify get orders.acme.cert-manager.io -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/acme-orders.json || true
        kubectl --insecure-skip-tls-verify get statefulsets -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/statefulsets.json || true
        kubectl --insecure-skip-tls-verify get secrets -n $NAMESPACE -o json |jq 'del(.items[].data)' 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/secrets.json || true
        kubectl --insecure-skip-tls-verify get certificates -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/certificates.json || true
        kubectl --insecure-skip-tls-verify get MetricsTrait -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/metrics-traits.json || true
        kubectl --insecure-skip-tls-verify get servicemonitor -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/service-monitors.json || true
        kubectl --insecure-skip-tls-verify get podmonitor -n $NAMESPACE -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/$NAMESPACE/pod-monitors.json || true
      fi
    fi
  done <$CAPTURE_DIR/cluster-snapshot/namespace_list.out
  rm $CAPTURE_DIR/cluster-snapshot/namespace_list.out
}

function full_k8s_cluster_snapshot() {
  echo "Full capture of kubernetes cluster"
  # Get general cluster-info dump, this contains quite a bit but not everything, it also sets up the directory structure
  kubectl --insecure-skip-tls-verify cluster-info dump --all-namespaces --output-directory=$CAPTURE_DIR/cluster-snapshot >/dev/null 2>&1

  # Get the Verrazzano resource at the root level. The Verrazzano custom resource can define the namespace, so use all the namespaces in the command
  kubectl --insecure-skip-tls-verify get verrazzano --all-namespaces -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/verrazzano-resources.json || true

  if [ $? -eq 0 ]; then
    kubectl --insecure-skip-tls-verify version -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/kubectl-version.json || true
    kubectl --insecure-skip-tls-verify get crd -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/crd.json || true
    kubectl --insecure-skip-tls-verify get pv -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/pv.json || true
    kubectl --insecure-skip-tls-verify get ingress -A -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/ingress.json || true
    kubectl --insecure-skip-tls-verify api-resources -o wide 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/api_resources.out || true
    kubectl --insecure-skip-tls-verify get netpol -A -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/network-policies.json || true
    kubectl --insecure-skip-tls-verify describe netpol -A 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/network-policies.txt || true
    kubectl --insecure-skip-tls-verify describe ClusterIssuer -A 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/cluster-issuers.txt || true
    kubectl --insecure-skip-tls-verify get MutatingWebhookConfigurations -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/mutating-webhook-configs.txt || true
    kubectl --insecure-skip-tls-verify get ValidatingWebhookConfigurations -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/validating-webhook-configs.txt || true
    # squelch the "too many clients" warnings from newer kubectl versions
    dump_extra_details_per_namespace
    dump_configmaps
    helm version 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/helm-version.out || true
    helm ls -A -o json 2>/dev/null > $CAPTURE_DIR/cluster-snapshot/helm-ls.json || true
    dump_es_indexes > $CAPTURE_DIR/cluster-snapshot/es_indexes.out || true
    process_nodes_output || true
    # dump the Prometheus scrape configuration
    if kubectl get ns verrazzano-monitoring 2>&1 > /dev/null ; then
      kubectl get secret prometheus-prometheus-operator-kube-p-prometheus -n verrazzano-monitoring -o json | jq -r '.data["prometheus.yaml.gz"]' | base64 -d | gunzip > $CAPTURE_DIR/cluster-snapshot/prom-scrape-config.yaml || true
    fi
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
      cd $SCRIPT_DIR/../vz
      # To enable debug, add  -zap-log-level debug
      if [ -z $REPORT_FILE ]; then
          if [[ -x $GOPATH/bin/vz ]]; then
            $GOPATH/vz analyze --capture-dir $FULL_PATH_CAPTURE_DIR || true
          else
            GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go analyze --capture-dir $FULL_PATH_CAPTURE_DIR || true
          fi
      else
          # Since we have to change the current working directory to run go, we need to take into account if the reportFile specified was relative to the original
          # working directory. If it was absolute then we just use it directly
          if [[ $REPORT_FILE = /* ]]; then
              if [[ -x $GOPATH/bin/vz ]]; then
                  $GOPATH/vz analyze --capture-dir $FULL_PATH_CAPTURE_DIR --report-format detailed --report-file $REPORT_FILE || true
                else
                  GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go analyze --capture-dir $FULL_PATH_CAPTURE_DIR --report-format detailed --report-file $REPORT_FILE || true
              fi
            else
              if [[ -x $GOPATH/bin/vz ]]; then
                  $GOPATH/vz analyze --capture-dir $FULL_PATH_CAPTURE_DIR --report-format detailed --report-file $SAVE_DIR/$REPORT_FILE || true
                else
                  GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go analyze --capture-dir $FULL_PATH_CAPTURE_DIR --report-format detailed --report-file $SAVE_DIR/$REPORT_FILE || true
              fi
          fi
        fi
    fi
  cd $SAVE_DIR
  fi
}

function save_dump_file() {
  # This will save the dump to a tar gz file if that was specified
  if [ ! -z $TAR_GZ_FILE ]; then
    # We only save files into cluster-snapshot and below we do not save the temp directory portion
    if [ -d $CAPTURE_DIR/cluster-snapshot ]; then
      tar -czf $TAR_GZ_FILE -C $CAPTURE_DIR cluster-snapshot
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

full_k8s_cluster_snapshot
if [ $? -eq 0 ]; then
  save_dump_file
fi

analyze_dump
cleanup_dump
