#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function create-kubeconfig {
  # Get the name of the secret containing the certificate for accessing the cluster
  default_secret=$(kubectl get serviceaccount default -o json | jq -r '.secrets[].name')

  # Get the certificate for accessing the kubernetes cluster
  default_cert=$(kubectl get secret $default_secret -o json | jq -r '.data."ca.crt"')

  # Get the endpoint for the kubernetes master server
  # The sed command is to strip out color escape sequences
  master_server=$(kubectl cluster-info | grep master | awk '{ print $6 }' | sed $'s/\e\\[[0-9;:]*[a-zA-Z]//g' )

  # Create a kubeconfig for the pod
  cp /verrazzano/config/kubeconfig-template $VERRAZZANO_KUBECONFIG
  sed -i -e "s|CERTIFICATE|$default_cert|g" -e "s|SERVER_ADDRESS|$master_server|g" $VERRAZZANO_KUBECONFIG
  export KUBECONFIG=$VERRAZZANO_KUBECONFIG
}

echo "*************************************************************"
echo " Running in ${MODE} mode                                     "
echo "*************************************************************"

if [ -n "${VERRAZZANO_KUBECONFIG}" ]; then
  # Set up a valid Kubeconfig for tools that require them
  create-kubeconfig
fi

# Run the operator
/usr/local/bin/verrazzano-platform-operator $*
