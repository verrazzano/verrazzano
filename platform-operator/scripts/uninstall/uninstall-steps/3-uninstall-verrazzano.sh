#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..
MANIFESTS_DIR=$SCRIPT_DIR/../../../thirdparty/manifests

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

VERRAZZANO_NS=verrazzano-system
VERRAZZANO_MONITORING_NS=verrazzano-monitoring

function delete_verrazzano() {
  # delete helm installation of Verrazzano
  # - specifically delete the verrazzano-system/verrazzano chart, since it's possible the
  #   verrazzano-platform-operator might get installed via helm separately
  log "Deleting Verrazzano"
  helm ls -n verrazzano-system \
    | awk '/verrazzano/ {print $1}' \
    | xargsr helm uninstall -n verrazzano-system \
    || err_return $? "Could not delete Verrazzano from helm" || return $? # return on pipefail

  # delete verrazzano-managed-cluster-local secret
  log "Deleting Verrazzano secrets"
  kubectl delete secret verrazzano-managed-cluster-local --ignore-not-found=true || err_return $? "Could not delete secrets from Verrazzano" || return $?

  log "Deleting ClusterRoleBindings"
  # deleting clusterrolebindings
  delete_k8s_resources clusterrolebinding ":metadata.name,:metadata.labels" "Could not delete ClusterRoleBindings from Verrazzano" '/verrazzano/ && ! /verrazzano-platform-operator/ && ! /verrazzano-install/ && ! /verrazzano-managed-cluster/ {print $1}' \
    || return $? # return on pipefail

  # deleting clusterroles
  log "Deleting ClusterRoles"
  delete_k8s_resources clusterrole ":metadata.name,:metadata.labels" "Could not delete ClusterRoles from Verrazzano" '/verrazzano/ && ! /verrazzano-managed-cluster/ {print $1}' \
    || return $? # return on pipefail

  # deleting namespaces
  log "Deleting Verrazzano namespace finalizers"
  # delete namespace finalizers
  patch_k8s_resources namespace ":metadata.name,:metadata.labels" "Could not remove finalizers from Verrazzano namespaces" '/k8s-app:verrazzano.io|verrazzano.io\/namespace:monitoring|verrazzano-system|verrazzano-mc/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting Verrazzano namespaces"
  delete_k8s_resources namespace ":metadata.name,:metadata.labels" "Could not delete Verrazzano namespaces" '/k8s-app:verrazzano.io|verrazzano.io\/namespace:monitoring|verrazzano-system/ {print $1}' \
    || return $? # return on pipefail

  # Delete CR'S from all Verrazzano managed namespaces
  delete_managed_k8s_resources healthscopes.core.oam.dev
  delete_managed_k8s_resources manualscalertraits.core.oam.dev
  delete_managed_k8s_resources traitdefinitions.core.oam.dev
  delete_managed_k8s_resources workloaddefinitions.core.oam.dev
  delete_managed_k8s_resources scopedefinitions.core.oam.dev
}

function delete_weblogic_operator {
  log "Uninstall the WebLogic Kubernetes operator"
  if helm status uninstall weblogic-operator --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall weblogic-operator --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall the WebLogic Kubernetes operator."
    fi
  fi


}

function delete_kiali {
  KIALI_CHART_DIR=${CHARTS_DIR}/kiali-server
  log "Uninstall Kiali"
  if helm status kiali-server  --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall kiali-server  --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall Kiali."
    fi
  fi
  log "Deleting Kiali Custom Resource Definitions"
  kubectl delete -f ${KIALI_CHART_DIR}/crds || true
}

action "Deleting Verrazzano Components" delete_verrazzano || exit 1
action "Deleting Kiali " delete_kiali || exit 1
