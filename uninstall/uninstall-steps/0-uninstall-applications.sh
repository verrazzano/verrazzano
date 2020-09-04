#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

function initializing_uninstall {
  # Deleting rancher through API
  log "Deleting Rancher through API"
  rancher_exists=$(kubectl get namespace cattle-system) || return 0
  rancher_host_name="$(kubectl get ingress -n cattle-system --no-headers -o custom-columns=":spec.rules[0].host")" || return $?
  rancher_cluster_url="https://${rancher_host_name}/v3/clusters/local"
  rancher_admin_password=$(kubectl get secret --namespace cattle-system rancher-admin-secret -o jsonpath={.data.password}) || return $?
  rancher_admin_password=$(echo ${rancher_admin_password} | base64 --decode) || return $?

  if [ "$rancher_admin_password" ] && [ "$rancher_host_name" ] ; then
    echo "Get Rancher access token."
    get_rancher_access_token "${rancher_host_name}" "${rancher_admin_password}"
  fi

  if [ "${RANCHER_ACCESS_TOKEN}" ]; then
    log "Updating ${rancher_cluster_url}"
    status=$(curl -o /dev/null -s -w "%{http_code}\n" -X DELETE -H "Accept: application/json" -H "Authorization: Bearer ${RANCHER_ACCESS_TOKEN}" --insecure "${rancher_cluster_url}")
    if [ "$status" != 200 ] && [ "$status" != 404 ] ; then
      return 1
    fi
    while [ true ] ; do
      still_exists="$(curl -s -X GET -H "Accept: application/json" -H "Authorization: Bearer ${RANCHER_ACCESS_TOKEN}" --insecure "${rancher_cluster_url}")"
      state="$(echo "$still_exists" | jq -r ".state" )"
      if [ "$state" != "active" ] && [ "$state" != "removing" ] ; then
        break
      else
        log "Rancher cluster is still in state: ${state}"
        sleep 10
      fi
    done
  fi
}

function delete_bindings {
  kubectl get crd verrazzanobindings.verrazzano.io || return 0
  kubectl get VerrazzanoBindings --no-headers -o custom-columns=":metadata.name" \
    | xargsr kubectl delete VerrazzanoBindings \
    || return $? # return on pipefail
}

function delete_models {
  kubectl get crd verrazzanomodels.verrazzano.io || return 0
  kubectl get VerrazzanoModels --no-headers -o custom-columns=":metadata.name" \
    | xargsr kubectl delete VerrazzanoModels \
    || return $? # return on pipefail
}

action "Initializing Uninstall" initializing_uninstall || exit 1
action "Deleting Verrazzano Bindings" delete_bindings || exit 1
action "Deleting Verrazzano Models" delete_models || exit 1