#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
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

function delete_verrazzano() {
  # delete helm installation of Verrazzano
  log "Deleting Verrazzano"
  helm ls -A \
    | awk '/verrazzano/ {print $1}' \
    | xargsr helm uninstall -n verrazzano-system \
    || err_return $? "Could not delete verrazzano from helm" || return $? # return on pipefail

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

  # Delete CRDS from all namespaces
  delete_k8s_resource_from_all_namespaces applicationconfigurations.core.oam.dev
  delete_k8s_resource_from_all_namespaces coherence.coherence.oracle.com
  delete_k8s_resource_from_all_namespaces components.core.oam.dev
  delete_k8s_resource_from_all_namespaces containerizedworkloads.core.oam.dev
  delete_k8s_resource_from_all_namespaces domains.weblogic.oracle
  delete_k8s_resource_from_all_namespaces healthscopes.core.oam.dev
  delete_k8s_resource_from_all_namespaces manualscalertraits.core.oam.dev
  delete_k8s_resource_from_all_namespaces traitdefinitions.core.oam.dev
  delete_k8s_resource_from_all_namespaces workloaddefinitions.core.oam.dev
  delete_k8s_resource_from_all_namespaces scopedefinitions.core.oam.dev
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

action "Deleting Verrazzano Application Kubernetes operator" delete_application_operator || exit 1
action "Deleting OAM Kubernetes operator" delete_oam_operator || exit 1
action "Deleting Coherence Kubernetes operator" delete_coherence_operator || exit 1
action "Deleting WebLogic Kubernetes operator" delete_weblogic_operator || exit 1
action "Deleting Verrazzano Components" delete_verrazzano || exit 1