#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

UPDATE_OVERRIDE_CONFIGMAP_FILE=$SCRIPT_DIR/../../config/scripts/post-install-overrides/test-overrides-configmap.yaml
UPDATE_OVERRIDE_SECRET_FILE=$SCRIPT_DIR/../../config/scripts/post-install-overrides/test-overrides-secret.yaml

echo "Update overrides ConfigMap"
kubectl create cm test-overrides --from-file=$UPDATE_OVERRIDE_CONFIGMAP_FILE --dry-run=client | kubectl apply -f -
if [ $? -ne 0 ]; then
  echo "Could not update ConfigMap"
  exit 1
fi

echo "Update overrides Secret"
kubectl create secret generic test-overrides --from-file=$UPDATE_OVERRIDE_SECRET_FILE --dry-run=client | kubectl apply -f -
if [ $? -ne 0 ]; then
  echo "Could not update Secret"
  exit 1
fi