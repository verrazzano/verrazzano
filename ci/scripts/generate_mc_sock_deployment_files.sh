#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Creates modified versions of the sock shop MC example

if [ -z "$MC_SS_SOURCE_DIR" ] || [ -z "$MC_SS_DEST_DIR" ] || [ -z "$MC_APP_NAMESPACE" ] || [ -z "$MC_PROJ_NAME" ]; then
  echo "Variables MC_SS_SOURCE_DIR, MC_SS_DEST_DIR, MC_APP_NAMESPACE and MC_PROJ_NAME must be specified to run this script."
fi

# create output dir
mkdir -p $MC_SS_DEST_DIR

# create project file
yq eval ".metadata.name"=\"${MC_PROJ_NAME}\" $MC_SS_SOURCE_DIR/verrazzano-project.yaml > $MC_SS_DEST_DIR/verrazzano-project.yaml
yq -i eval ".spec.template.namespaces[0].metadata.name"=\"${MC_APP_NAMESPACE}\" $MC_SS_DEST_DIR/verrazzano-project.yaml

# create component file
yq eval ".spec.workload.metadata.namespace"=\"${MC_APP_NAMESPACE}\" $MC_SS_SOURCE_DIR/sock-shop-comp.yaml > $MC_SS_DEST_DIR/sock-shop-comp.yaml
yq -i eval ".metadata.namespace"=\"${MC_APP_NAMESPACE}\" $MC_SS_DEST_DIR/sock-shop-comp.yaml

# create MC app config file
yq eval ".metadata.namespace"=\"${MC_APP_NAMESPACE}\" $MC_SS_SOURCE_DIR/sock-shop-app.yaml > $MC_SS_DEST_DIR/sock-shop-app.yaml
