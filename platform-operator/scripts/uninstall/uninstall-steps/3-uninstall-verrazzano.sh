#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

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

  # delete crds
  log "Deleting Verrazzano crd finalizers"
  patch_k8s_resources crds ":metadata.name" "Could not remove finalizers from CustomResourceDefinitions in Verrazzano" '/verrazzano.io/' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting Verrazzano crds"
  delete_k8s_resources crds ":metadata.name" "Could not delete CustomResourceDefinitions from Verrazzano" '/verrazzano.io/ && ! /verrazzanos.install.verrazzano.io/ && ! /verrazzanomanagedclusters.clusters.verrazzano.io/' \
    || return $? # return on pipefail

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
  delete_k8s_resources namespace ":metadata.name,:metadata.labels" "Could not delete Verrazzano namespaces" '/k8s-app:verrazzano.io|verrazzano.io\/namespace:monitoring|verrazzano-system|verrazzano-mc/ {print $1}' \
    || return $? # return on pipefail

  # Delete CR'S from all Verrazzano managed namespaces
  delete_managed_k8s_resources applicationconfigurations.core.oam.dev
  delete_managed_k8s_resources coherence.coherence.oracle.com
  delete_managed_k8s_resources components.core.oam.dev
  delete_managed_k8s_resources containerizedworkloads.core.oam.dev
  delete_managed_k8s_resources domains.weblogic.oracle
  delete_managed_k8s_resources healthscopes.core.oam.dev
  delete_managed_k8s_resources manualscalertraits.core.oam.dev
  delete_managed_k8s_resources traitdefinitions.core.oam.dev
  delete_managed_k8s_resources workloaddefinitions.core.oam.dev
  delete_managed_k8s_resources scopedefinitions.core.oam.dev
}

function delete_oam_operator {
  log "Uninstall the OAM Kubernetes operator"
  if helm status oam-kubernetes-runtime --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall oam-kubernetes-runtime --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall the OAM Kubernetes operator."
    fi
  fi
}

function delete_application_operator {
  log "Uninstall the Verrazzano Kubernetes application operator"
  if helm status verrazzano-application-operator --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall verrazzano-application-operator --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall the Verrazzano Kubernetes application operator."
    fi
  fi
}

function delete_authproxy {
  log "Uninstall the Verrazzano AuthProxy"
  if helm status verrazzano-authproxy --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall verrazzano-authproxy --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall the Verrazzano AuthProxy."
    fi
  fi
}



function delete_weblogic_operator {
  log "Uninstall the WebLogic Kubernetes operator"
  if helm status uninstall weblogic-operator --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall weblogic-operator --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall the WebLogic Kubernetes operator."
    fi
  fi

  log "Delete the WebLogic Kubernetes operator service account"
  if kubectl get serviceaccount -n "${VERRAZZANO_NS}" weblogic-operator-sa > /dev/null 2>&1 ; then
    if ! kubectl delete serviceaccount -n "${VERRAZZANO_NS}" weblogic-operator-sa ; then
      error "Failed to delete the WebLogic Kubernetes operator service account."
    fi
  fi
}

function delete_coherence_operator {
  log "Uninstall the Coherence Kubernetes operator"
  if helm status uninstall coherence-operator --namespace "${VERRAZZANO_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall coherence-operator --namespace "${VERRAZZANO_NS}" ; then
      error "Failed to uninstall the Coherence Kubernetes operator."
    fi
  fi
  kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io coherence-operator-validating-webhook-configuration --ignore-not-found
  kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io coherence-operator-mutating-webhook-configuration --ignore-not-found
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

function delete_prometheus_adapter {
  log "Uninstall the Prometheus adapter"
  if helm status prometheus-adapter --namespace "${VERRAZZANO_MONITORING_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall prometheus-adapter --namespace "${VERRAZZANO_MONITORING_NS}" ; then
      error "Failed to uninstall the Prometheus adapter."
    fi
  fi
}

function delete_prometheus_operator {
  log "Uninstall the Prometheus operator"
  if helm status prometheus-operator --namespace "${VERRAZZANO_MONITORING_NS}" > /dev/null 2>&1 ; then
    if ! helm uninstall prometheus-operator --namespace "${VERRAZZANO_MONITORING_NS}" ; then
      error "Failed to uninstall the Prometheus operator."
    fi
  fi

  log "Deleting ${VERRAZZANO_MONITORING_NS} namespace finalizers"
  patch_k8s_resources namespace ":metadata.name" "Could not remove finalizers from namespace ${VERRAZZANO_MONITORING_NS}" "/${VERRAZZANO_MONITORING_NS}/ {print \$1}" '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting the ${VERRAZZANO_MONITORING_NS} namespace"
  kubectl delete namespace "${VERRAZZANO_MONITORING_NS}" --ignore-not-found=true || err_return $? "Could not delete the ${VERRAZZANO_MONITORING_NS} namespace"
}

action "Deleting Prometheus adapter " delete_prometheus_adapter || exit 1
action "Deleting Prometheus operator " delete_prometheus_operator || exit 1
action "Deleting Verrazzano Application Kubernetes operator" delete_application_operator || exit 1
action "Deleting OAM Kubernetes operator" delete_oam_operator || exit 1
action "Deleting Coherence Kubernetes operator" delete_coherence_operator || exit 1
action "Deleting WebLogic Kubernetes operator" delete_weblogic_operator || exit 1
action "Deleting Verrazzano AuthProxy" delete_authproxy || exit 1
action "Deleting Verrazzano Components" delete_verrazzano || exit 1
action "Deleting Kiali " delete_kiali || exit 1
