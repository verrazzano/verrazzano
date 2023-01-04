#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function create-kubeconfig {
  # Create a kubeconfig file for the pod. The kubeconfig is needed by the Rancher uninstall tool so we create a
  # bare minimum kubeconfig to satisfy Rancher.
  cp /verrazzano/config/kubeconfig-template $VERRAZZANO_KUBECONFIG
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
