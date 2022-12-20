#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function create-kubeconfig {
  # Get the endpoint for the kubernetes master server
  # The sed command is to strip out color escape sequences
  master_server=$(kubectl cluster-info | grep "control plane" | awk '{ print $7 }' | sed $'s/\e\\[[0-9;:]*[a-zA-Z]//g' )

  # Create a kubeconfig for the pod
  cp /verrazzano/config/kubeconfig-template $VERRAZZANO_KUBECONFIG
  sed -i -e "s|SERVER_ADDRESS|$master_server|g" $VERRAZZANO_KUBECONFIG
  export KUBECONFIG=$VERRAZZANO_KUBECONFIG
  chmod 600 ${KUBECONFIG}
  ls -l ${KUBECONFIG}
}

if [ -n "${VERRAZZANO_KUBECONFIG}" ]; then
  # If VERRAZZANO_KUBECONFIG is set, set up a valid Kubeconfig for tools that require them at the requested location
  create-kubeconfig
fi

# Run the operator
/usr/local/bin/verrazzano-platform-operator $*
