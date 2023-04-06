#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function create-kubeconfig {
    # Get the name of the secret containing the certificate for accessing the cluster
    secret=$(kubectl get serviceAccount verrazzano-platform-operator -n verrazzano-install -o=jsonpath='{.secrets[0].name}')

    # Get the certificate for accessing the kubernetes cluster
    ca=$(kubectl get secret $secret -n verrazzano-install -o=jsonpath='{.data.ca\.crt}')

    # Get the user token
    token=$(kubectl get secret $secret -n verrazzano-install -o=jsonpath='{.data.token}' | base64 --decode)

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
