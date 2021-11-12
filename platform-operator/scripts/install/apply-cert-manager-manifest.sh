#!/bin/bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

# create kubeconfig before importing common
function create-kubeconfig {
  # Get the name of the secret containing the certificate for accessing the cluster
  default_secret=$(kubectl get serviceaccount default -o json | jq -r '.secrets[].name')

  # Get the certificate for accessing the kubernetes cluster
  default_cert=$(kubectl get secret $default_secret -o json | jq -r '.data."ca.crt"')

  # Get the endpoint for the kubernetes master server
  # The sed command is to strip out color escape sequences
  master_server=$(kubectl cluster-info | grep master | awk '{ print $6 }' | sed $'s/\e\\[[0-9;:]*[a-zA-Z]//g' )

  # Create a kubeconfig for the pod
  cp /verrazzano/config/kubeconfig-template $VERRAZZANO_KUBECONFIG
  sed -i -e "s|CERTIFICATE|$default_cert|g" -e "s|SERVER_ADDRESS|$master_server|g" $VERRAZZANO_KUBECONFIG
  export KUBECONFIG=$VERRAZZANO_KUBECONFIG
}
create-kubeconfig

. ${SCRIPT_DIR}/logging.sh
. $SCRIPT_DIR/common.sh

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

function setup_cert_manager_crd() {
  local CERT_MANAGER_MANIFEST_DIR=${MANIFESTS_DIR}/cert-manager
  cp "$CERT_MANAGER_MANIFEST_DIR/cert-manager.crds.yaml" "$TMP_DIR/cert-manager.crds.yaml"
  if [ "$DNS_TYPE" == "oci" ]; then
    command -v patch >/dev/null 2>&1 || {
      fail "patch is required but cannot be found on the path. Aborting.";
    }
    log "Patching cert-manager.crds.yaml to add OCI DNS"
    patch "$TMP_DIR/cert-manager.crds.yaml" "$SCRIPT_DIR/config/cert-manager.crds.patch"
  fi

  yaml=$(<"$TMP_DIR/cert-manager.crds.yaml")
  kubectl_apply_with_retry "$yaml" --validate=false
}

function kubectl_apply_with_retry() {
  local count=0
  local ret=0
  until kubectl apply -f <(echo "$1") "${@:2}"; do
    ret=$?
    count=$((count+1))
    if [[ "$count" -lt 60 ]]; then
      echo "kubectl apply failed, waiting for 5 seconds and trying again"
      sleep 5
    else
      echo "kubectl apply attempt timed out."
      break
    fi
  done

  if [ $ret -ne 0 ]; then
    echo "kubectl apply failed with non-zero return code."
  else
    echo "kubectl apply succeeded."
  fi
  return $ret
}

setup_cert_manager_crd