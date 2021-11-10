#!/bin/bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. ${SCRIPT_DIR}/logging.sh

CONFIG_DIR=$SCRIPT_DIR/config
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
}

setup_cert_manager_crd
yaml=$(<"$TMP_DIR/cert-manager.crds.yaml")
kubectl_apply_with_retry "$yaml" --validate=false