#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
export DIRECTORY="/Users/sdosapat/Downloads/pabhat%2Fvz-6801-last-clean-periodic-test_verrazzano-1.4.0-open-source/verrazzano-1.4.0-linux-amd64"
cd ${DIRECTORY}

if [ -e "LICENSE" ]
then
  echo 'License found'
else
  echo 'ERROR: Missing LICENSE file'
  exit 1
fi

cd ${DIRECTORY}/bin
if [ -e "bom_utils.sh" ] && [ -e "vz" ] && [ -e "vz-registry-image-helper.sh" ]
then
  echo 'All files found under bin'
else
  echo 'ERROR: Missing files for bin'
  exit 1
fi

cd ${DIRECTORY}/manifests
if [ -e "verrazzano-bom.json" ]
then
  echo 'All files found under manifests'
else
  echo 'ERROR: Missing files for manifests'
  exit 1
fi

if [ -e "k8s/verrazzano-platform-operator.yaml" ]
then
  echo 'All files found under manifests/k8s'
else
  echo 'ERROR: Missing files for manifests/k8s'
  exit 1
fi

if [ -e "profiles/default.yaml" ] && [ -e "profiles/dev.yaml" ] && [ -e "profiles/managed-cluster.yaml" ] && [ -e "profiles/oci.yaml" ] && [ -e "profiles/ocne.yaml" ]
then
  echo 'All files found under manifests/profiles'
else
  echo 'ERROR: Missing files for manifests/profiles'
  exit 1
fi

cd ${DIRECTORY}/manifests/charts/verrazzano-platform-operator
if [ -e "Chart.yaml" ] && [ -e "NOTES.txt" ] && [ -e "values.yaml" ]
then
  echo 'All files found under manifests/charts/verrazzano-platform-operator'
else
  echo 'ERROR: Missing files for manifests/charts/verrazzano-platform-operator'
  exit 1
fi

if [ -e "crds/clusters.verrazzano.io_verrazzanomanagedclusters.yaml" ] && [ -e "crds/install.verrazzano.io_verrazzanos.yaml" ]
then
  echo 'All files found under manifests/charts/verrazzano-platform-operator/crds'
else
  echo 'ERROR: Missing files for manifests/charts/verrazzano-platform-operator/crds'
  exit 1
fi

if [ -e "templates/clusterrole.yaml" ] && [ -e "templates/clusterrolebinding.yaml" ] && [ -e "templates/deployment.yaml" ] && [ -e "templates/namespace.yaml" ] && \
    [ -e "templates/service.yaml" ] && [ -e "templates/serviceaccount.yaml" ] && [ -e "templates/validatingwebhookconfiguration.yaml" ]
then
  echo 'All files found under manifests/charts/verrazzano-platform-operator/bin'
else
  echo 'ERROR: Missing files for manifests/charts/verrazzano-platform-operator/bin'
  exit 1
fi
