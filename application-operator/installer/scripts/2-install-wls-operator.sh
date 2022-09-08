#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh
TMP_DIR=$(mktemp -d)

function install_wls_operator {
  log "Clone WebLogic Kubernetes operator"
  git clone https://github.com/oracle/weblogic-kubernetes-operator.git $TMP_DIR/weblogic-kubernetes-operator
  if [ $? -ne 0 ]; then
    error "Failed to clone WebLogic Kubernetes operator."
    return 1
  fi

  log "Checkout WebLogic Kubernetes operator"
  git --git-dir=$TMP_DIR/weblogic-kubernetes-operator/.git --work-tree=$TMP_DIR/weblogic-kubernetes-operator checkout v3.4.3
  if [ $? -ne 0 ]; then
    error "Failed to checkout WebLogic Kubernetes operator."
    return 1
  fi

  log "Create weblogic-operator serviceaccount"
  kubectl create serviceaccount -n verrazzano-system weblogic-operator-sa
  if [ $? -ne 0 ]; then
    error "Failed to create weblogic-operator serviceaccount."
    return 1
  fi

  log "Install WebLogic Kubernetes operator"
  helm install weblogic-operator $TMP_DIR/weblogic-kubernetes-operator/kubernetes/charts/weblogic-operator --namespace verrazzano-system --set image=ghcr.io/oracle/weblogic-kubernetes-operator:3.4.3 --set serviceAccount=weblogic-operator-sa --set domainNamespaceSelectionStrategy=LabelSelector --set domainNamespaceLabelSelector=verrazzano-managed --set enableClusterRoleBinding=true --wait
  if [ $? -ne 0 ]; then
    error "Failed to install WebLogic Kubernetes operator."
    return 1
  fi
}

action "Installing WebLogic Kubernetes operator" install_wls_operator || fail "Failed to install WebLogic Kubernetes operator."
