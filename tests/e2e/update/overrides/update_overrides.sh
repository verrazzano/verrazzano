#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
CM_FILE=$SCRIPT_DIR
SECRET_FILE=$SCRIPT_DIR

CREATE_OVERRIDE_CONFIGMAP_FILE=$SCRIPT_DIR/../../config/scripts/pre-install-overrides/test-overrides-configmap.yaml
CREATE_OVERRIDE_SECRET_FILE=$SCRIPT_DIR/../../config/scripts/pre-install-overrides/test-overrides-secret.yaml
UPDATE_OVERRIDE_CONFIGMAP_FILE=$SCRIPT_DIR/../../config/scripts/post-install-overrides/test-overrides-configmap.yaml
UPDATE_OVERRIDE_SECRET_FILE=$SCRIPT_DIR/../../config/scripts/post-install-overrides/test-overrides-secret.yaml

if [[ "$1" == "Create" ]]; then
  CM_FILE=$CREATE_OVERRIDE_CONFIGMAP_FILE
  SECRET_FILE=$CREATE_OVERRIDE_SECRET_FILE
elif [[ "$1" == "Update" ]]; then
  CM_FILE=$UPDATE_OVERRIDE_CONFIGMAP_FILE
  SECRET_FILE=$UPDATE_OVERRIDE_SECRET_FILE
fi

echo "Create/Update overrides ConfigMap"
kubectl create cm test-overrides --from-file=$CM_FILE -o yaml --dry-run=client | kubectl apply -f -
if [ $? -ne 0 ]; then
  echo "Could not update ConfigMap"
  exit 1
fi

echo "Create/Update overrides Secret"
kubectl create secret generic test-overrides --from-file=$SECRET_FILE -o yaml --dry-run=client | kubectl apply -f -
if [ $? -ne 0 ]; then
  echo "Could not update Secret"
  exit 1
fi

