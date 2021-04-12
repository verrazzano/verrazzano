#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
VERRAZZANO_NS=verrazzano-system
VERRAZZANO_API=verrazzano-api
SERVICE_ACCOUNT=ServiceAccount
EXPECTED_SUBJECT="kind:$SERVICE_ACCOUNT name:$VERRAZZANO_API namespace:$VERRAZZANO_NS"
EXPECTED_RESOURCES_VERBS="resources:[users groups] verbs:[impersonate]"
TEST_FAILURE=false

# Validate the secret of the ServiceAccount
function validate_secret_service_account() {
  # Get API Proxy pod name
  local api_proxi_pod_name="$(kubectl get pods -l app=$VERRAZZANO_API -n $VERRAZZANO_NS \
    -o 'jsonpath={.items[0].metadata.name}')"
  if [[ -z "${api_proxi_pod_name}" ]]; then
    echo "Failure to get $VERRAZZANO_API pod name."
    exit 1
  fi

  # Get secret for the SA
  local sa_secret="$(kubectl -n $VERRAZZANO_NS get serviceaccount $VERRAZZANO_API \
    -o 'jsonpath={.secrets[0].name}')"
  if [[ -z "${sa_secret}" ]]; then
    echo "Failure to get the secret for the ServiceAccount $VERRAZZANO_API."
    exit 1
  fi

  #Get API proxy pod and look for the secret
  local secret_name_from_pod="$(kubectl get pod $api_proxi_pod_name -n $VERRAZZANO_NS \
    -o "jsonpath={.spec.volumes[?(@.name==\"${sa_secret}\")].secret.secretName}")"
  if [[ -z "${secret_name_from_pod}" ]]; then
    echo "Failure to get the secret from the pod."
    exit 1
  fi

  if [ "${sa_secret}" != "${secret_name_from_pod}" ]; then
    echo "FAIL: The secret name of ServiceAccount $VERRAZZANO_API differs from the secret obtained from the pod."
    TEST_FAILURE=true
  else
    echo "PASS: Validation of the secret for the ServiceAccount $VERRAZZANO_API succeeded."
  fi
}

# Validate the subject of the ClusterRoleBinding(s) and impersonate role contains only the expected
# impersonate permissions.
function validate_role_binding_service_account() {
  #Get cluster role bindings for verrazzano-api and extract the cluster roles
  local cluster_role_name_sa="$(kubectl get clusterrolebindings -n $VERRAZZANO_NS \
    -o custom-columns="NAME:metadata.name,SERVICE_ACCOUNTS:subjects[?(@.kind==\"$SERVICE_ACCOUNT\")].name" \
    | grep "${VERRAZZANO_API}" | awk {'print $1'})"
    if [[ -z "${cluster_role_name_sa}" ]]; then
    echo "Failure to get the role binding for the Service Account."
    exit 1
  fi

  # cluster_role_name_sa might contain multiple values, separated by whitespace, split it in parts
  for c_role_binding in $cluster_role_name_sa
  do
    local cluster_role_subject="$(kubectl get clusterrolebinding $c_role_binding -n $VERRAZZANO_NS \
      -o "jsonpath={.subjects}")"
    if [[ "$cluster_role_subject" =~ .*"$EXPECTED_SUBJECT".* ]]; then
      echo "PASS: The ClusterRoleBinding $c_role_binding contains the expected Subject: [$EXPECTED_SUBJECT]"
    else
      echo "FAIL: The ClusterRoleBinding $c_role_binding does not contain the expected Subject: [$EXPECTED_SUBJECT]."
      TEST_FAILURE=true
    fi

    local c_role="$(kubectl get clusterrolebinding $c_role_binding -n $VERRAZZANO_NS -o "jsonpath={.roleRef.name}")"
    local cluster_role_resources_verbs="$(kubectl get clusterrole $c_role -n $VERRAZZANO_NS -o "jsonpath={.rules}")"
    if [[ "$cluster_role_resources_verbs" =~ .*"$EXPECTED_RESOURCES_VERBS".* ]]; then
      echo "PASS: The ClusterRole $c_role contains the expected Resources and Verbs: [$EXPECTED_RESOURCES_VERBS]."
    else
      echo "FAIL: The ClusterRole $c_role does not contain the expected Resources and Verbs: [$EXPECTED_RESOURCES_VERBS]."
      TEST_FAILURE=true
    fi
  done
}

validate_secret_service_account
validate_role_binding_service_account

if [ "$TEST_FAILURE" == true ] ; then
  echo "Verrazzano API Proxy validation failed."
  exit 1
fi

