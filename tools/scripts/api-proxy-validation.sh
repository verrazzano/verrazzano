#!/bin/bash -x
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
VERRAZZANO_NS=verrazzano-system
VERRAZZANO_API=verrazzano-api
SERVICE_ACCOUNT=ServiceAccount

function get_roleRef_name_for_subjects() {
    kubectl get clusterrolebinding -o json | jq -r "
      .items[]
      |
      select(
        .subjects[]?
        |
        select(
            .kind == \"${SERVICE_ACCOUNT}\"
            and
            .name == \"${VERRAZZANO_API}\"
            and
            (if .namespace then .namespace else \"\" end) == \"${VERRAZZANO_NS}\"
        )
      )
      |
      (.roleRef.name)
    "
}

function validate_api_proxy_service_account() {
  # Get API Proxy pod name
  local api_proxi_pod_name="$(kubectl get pods -l app=$VERRAZZANO_API -n $VERRAZZANO_NS -o 'jsonpath={.items[0].metadata.name}')"
  if [[ -z "${api_proxi_pod_name}" ]]; then
    echo "Failure to get $VERRAZZANO_API pod name."
    exit 1
  fi

  # Get secret for the SA
  local sa_secret="$(kubectl -n $VERRAZZANO_NS get serviceaccount $VERRAZZANO_API -o 'jsonpath={.secrets[0].name}')"
  if [[ -z "${sa_secret}" ]]; then
    echo "Failure to get the secret for the ServiceAccount $VERRAZZANO_API."
    exit 1
  fi

  #Get API proxy pod and look for the secret
  local secret_name_from_pod="$(kubectl get pod $api_proxi_pod_name -n $VERRAZZANO_NS -o "jsonpath={.spec.volumes[?(@.name==\"${sa_secret}\")].secret.secretName}")"
  if [[ -z "${secret_name_from_pod}" ]]; then
    echo "Failure to get the secret from the pod."
    exit 1
  fi
  #echo "Secret from the API proxy pod ${secret_name_from_pod}"

  if [ "${sa_secret}" != "${secret_name_from_pod}" ]; then
    echo "The secret name of ServiceAccount $VERRAZZANO_API differs from the secret obtained from the pod"
    exit 1
  fi

  #Get ClusterRoleBindings for verrazzano-api and describe
  local cluster_role_name_sa="$(kubectl get clusterrolebindings -n $VERRAZZANO_NS -o custom-columns="NAME:metadata.name,SERVICE_ACCOUNTS:subjects[?(@.kind==\"$SERVICE_ACCOUNT\")].name" | grep "${VERRAZZANO_API}" | awk {'print $1'})"
    if [[ -z "${cluster_role_name_sa}" ]]; then
    echo "Failure to get the role binding for the Service Account."
    exit 1
  fi
  cluster_role_name_sa="${cluster_role_name_sa}"
  cluster_role_name_using_subjects=$(get_roleRef_name_for_subjects)
    if [ "${cluster_role_name_sa}" != "${cluster_role_name_using_subjects}" ]; then
    echo "Failed to match the cluster role using based on the subjects Kind:ServiceAccount for $cluster_role_name_sa"
    exit 1
  fi
}

validate_api_proxy_service_account