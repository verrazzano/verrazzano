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

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CONFIG_DIR=$INSTALL_DIR/config

function finalize() {
  log "Deleting configmap istio-ca-root-cert from all namespaces"
  IFS=$'\n' read -r -d '' -a namespaces < <( kubectl get namespaces --no-headers -o custom-columns=":metadata.name" && printf '\0' )
  for ns in "${namespaces[@]}" ; do
    if kubectl get configmap istio-ca-root-cert > /dev/null 2>&1 ; then
      kubectl delete configmap istio-ca-root-cert --namespace ${ns} --ignore-not-found=true
    fi
  done

  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  local helm_ls
  helm_ls=$(helm repo ls >/dev/null 2>&1)
  if [ $? -eq 0 ]; then
    echo "$helm_ls" \
      | awk '/istio.io|stable|jetstack|rancher-stable|codecentric/ {print $1}' \
      | xargsr -I name helm repo remove name \
      || err_return $? "Could delete Helm Repos" || return $? # return on pipefail
  fi

}

# action "Finalizing Uninstall" finalize || exit 1
