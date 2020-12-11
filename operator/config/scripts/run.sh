#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
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

# Add installation logs to STDOUT so that they can be viewed after the job completes
function dump-install-logs {
  exitStatus=$1
  echo "**************************************************************"
  echo " Dumping the installation logs contained in install/build/logs"
  echo "**************************************************************"
  cat install/build/logs/*
  exit $exitStatus
}

# Add uninstall logs to STDOUT so that they can be viewed after the job completes
function dump-uninstall-logs {
  exitStatus=$1
  echo "*************************************************************"
  echo " Dumping the uninstall logs contained in uninstall/build/logs"
  echo "*************************************************************"
  cat uninstall/build/logs/*
  exit $exitStatus
}

# The same docker image is shared between the verrazzano-platform-operator and
# the installation jobs that the operator creates.  The default mode is to run
# the verrazzano-platform-operator.

# Create a kubeconfig
create-kubeconfig
if [ ${MODE} == "NOOP" ]; then
  echo "*************************************************************"
  echo " Running in NOOP mode, exiting                               "
  echo "*************************************************************"
  exit 0
elif [ ${MODE} == "INSTALL" ]; then
  # Run the installation
  ./install/1-install-istio.sh || dump-install-logs 1
  ./install/2-install-system-components.sh || dump-install-logs 1
  ./install/3-install-verrazzano.sh || dump-install-logs 1
  ./install/4-install-keycloak.sh || dump-install-logs 1
  dump-install-logs 0
elif [ ${MODE} == "UNINSTALL" ]; then
  # Run the uninstall
  ./uninstall/uninstall-verrazzano.sh -f || dump-uninstall-logs 1
  dump-uninstall-logs 0
else
  # Run the operator
  /usr/local/bin/verrazzano-platform-operator
fi
