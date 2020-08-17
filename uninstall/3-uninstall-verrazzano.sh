#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

function delete_verrazzano() {
  # delete helm installation of Verrazanno
  log "Deleting Verrazzano"
  helm delete verrazzano -n verrazzano-system || 2>/dev/null

  # delete verrazzano-managed-cluster-local secret
  if [ "$(kubectl get secret verrazzano-managed-cluster-local)" ] ; then
    kubectl delete secret verrazzano-managed-cluster-local
  fi

  # delete crds
  kubectl get crds --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano.io' \
    | xargs kubectl delete crd

  # deleting certificatesigningrequests
  kubectl get csr --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'csr-' \
    | xargs kubectl delete csr

  # deleting clusterrolebindings
  kubectl get clusterrolebinding --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'filebeat|journalbeat|node-exporter' \
    | xargs kubectl delete clusterrolebinding

  # deleting clusterroles
  kubectl get clusterrole --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'filebeat|journalbeat|node-exporter' \
    | xargs kubectl delete clusterrole

  # deleting namespaces
  kubectl get namespace --no-headers -o custom-columns=":metadata.name" \
    | grep -E 'verrazzano-system|monitoring|logging' \
    | xargs kubectl delete namespace
}

action "Deleting Verrazzano Components" delete_verrazzano