#!/bin/bash

# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

INSTALL_CONFIG_TO_EDIT=$1

# Thanos storage provider config that uses local filesystem
STORAGE_PROVIDER_CONFIG='type: FILESYSTEM
config:
  directory: "/tmp"
'

# Create the verrazzano-monitoring namespace if it does not exist and create a secret with the storage provider config
kubectl create ns verrazzano-monitoring 2>/dev/null || true
kubectl create secret generic -n verrazzano-monitoring objstore-config --from-literal=objstore.yml="${STORAGE_PROVIDER_CONFIG}"

# Modify the VZ CR to enable Thanos Store Gateway and to reference the storage provider secret
echo "Editing install config file for Thanos Store Gateway ${INSTALL_CONFIG_TO_EDIT}"
  yq -i eval ".spec.components.thanos.overrides.[0].values.existingObjstoreSecret = \"objstore-config\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.thanos.overrides.[0].values.storegateway.enabled = true" ${INSTALL_CONFIG_TO_EDIT}

# Modify the VZ CR to enable storage on the Prometheus Thanos Sidecar
echo "Editing install config file to enable long-term storage on the Prometheus Thanos Sidecar ${INSTALL_CONFIG_TO_EDIT}"
  yq -i eval ".spec.components.prometheusOperator.overrides.[2].values.prometheus.prometheusSpec.thanos.objectStorageConfig.key = \"objstore.yml\"" ${INSTALL_CONFIG_TO_EDIT}
  yq -i eval ".spec.components.prometheusOperator.overrides.[2].values.prometheus.prometheusSpec.thanos.objectStorageConfig.name = \"objstore-config\"" ${INSTALL_CONFIG_TO_EDIT}

cat ${INSTALL_CONFIG_TO_EDIT}
