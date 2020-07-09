#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

set -euo pipefail

echo "Removing Helidon hello world application."

if ! kubectl get vb -o name | grep verrazzanobinding.verrazzano.io/hello-world-binding; then
  echo "Delete application binding not required."
else
  echo "Delete application binding."
  if ! kubectl delete -f ${SCRIPT_DIR}/hello-world-binding.yaml --timeout 5m; then
    echo "ERROR: Delete application binding failed. Exiting."
    exit 1
  fi
fi

if ! kubectl get vm -o name | grep verrazzanomodel.verrazzano.io/hello-world-model; then
  echo "Delete application model not required."
else
  echo "Delete application model."
  if ! kubectl delete -f ${SCRIPT_DIR}/hello-world-model.yaml --timeout 2m; then
    echo "ERROR: Delete application model failed. Exiting."
    exit 1
  fi
fi

echo "Removal of Helidon hello world application was successful."
