#!/usr/bin/env bash
#
# Copyright (c) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

vpo="verrazzano-platform-operator"
namespace="verrazzano-install"

function create-kubeconfig {
  # Get the version of the Kubernetes server
  version=$(kubectl version -o json | jq -rj '.serverVersion|.major,".",.minor')

  # Secret does not exist for a serviceAccount starting with Kubernetes 1.24 so we create one.
  # The secret contains a certificate and token we need for accessing the cluster.
  if [[ "$version" > "1.23" ]]
  then
    kubectl apply -f /verrazzano/platform-operator/scripts/install/vpo-secret.yaml > /dev/null
    secret=$vpo
  else
    # Get the name of secret from the serviceAccount.
    secret=$(kubectl get serviceAccount $vpo -n $namespace -o=jsonpath='{.secrets[0].name}')
  fi

  # Get the certificate for accessing the kubernetes cluster
  ca=$(kubectl get secret $secret -n $namespace -o=jsonpath='{.data.ca\.crt}')

  # Get the user token
  token=$(kubectl get secret $secret -n $namespace -o=jsonpath='{.data.token}' | base64 --decode)

  # Get the endpoint for the kubernetes API server
  # The sed command is to strip out color escape sequences
  server=$(kubectl cluster-info | grep "control plane" | awk '{ print $7 }' | sed $'s/\e\\[[0-9;:]*[a-zA-Z]//g' )

  # Create a kubeconfig file for the pod.
  cp /verrazzano/config/kubeconfig-template $VERRAZZANO_KUBECONFIG
  sed -i -e "s|CA|$ca|g" -e "s|SERVER|$server|g" -e "s|TOKEN|$token|g" $VERRAZZANO_KUBECONFIG
  export KUBECONFIG=$VERRAZZANO_KUBECONFIG
  chmod 600 ${KUBECONFIG}
}

if [ -n "${VERRAZZANO_KUBECONFIG}" ]; then
  # If VERRAZZANO_KUBECONFIG is set, set up a valid Kubeconfig for tools that require them at the requested location
  create-kubeconfig
fi

# Run the operator
/usr/local/bin/verrazzano-platform-operator $*
