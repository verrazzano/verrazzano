#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)
BINDING="hello-world-binding"
MODEL="hello-world-model"

set -euo pipefail

echo "Removing Helidon hello world application."

# Delete the binding
echo "Delete application binding ${BINDING}."
if ! kubectl get VerrazzanoBindings ${BINDING}; then
  echo "Delete application binding not required. Skipping."
else
  # Ignore exit code since it isn't always correct
  kubectl delete -f ${SCRIPT_DIR}/hello-world-binding.yaml --timeout 5m || true
  # Check again to confirm that the binding still exists before failing.
  if kubectl get VerrazzanoBindings ${BINDING} &> /dev/null; then
    echo "ERROR: Delete application binding failed. Exiting."
    exit 1
  fi
fi

# Delete the model
echo "Delete application model ${MODEL}."
if ! kubectl get VerrazzanoModels ${MODEL}; then
  echo "Delete application model not required. Skipping."
else
  # Ignore exit code since it isn't always correct
  kubectl delete -f ${SCRIPT_DIR}/hello-world-model.yaml --timeout 2m || true
  # Check again to confirm that the model still exists before failing.
  if kubectl get VerrazzanoModels ${MODEL} &> /dev/null; then
    echo "ERROR: Delete application model failed. Exiting."
    exit 1
  fi
fi

echo "Removal of Helidon hello world application successful."
