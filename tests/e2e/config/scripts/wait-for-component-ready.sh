#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Waits for all component-level statuses to report Ready
#

resName=${1:-my-verrazzano}
components=($(kubectl get vz ${resName} -o json | jq -r '.status.components | keys[]'))

if [ ${#components[@]} -eq 0 ]; then
   echo "No components found for ${resName}"
   exit 1
fi

echo "Waiting for the following components to reach Ready state: ${components[@]}"

for comp in ${components[@]}; do
  echo "Checking component ${comp}"
  for iter in {1..20}; do
    state=$(kubectl  get vz my-verrazzano  -o jsonpath={.status.components.${comp}.state})
    echo "Component ${comp} state: ${state}"
    if [ "${state}" == "Disabled" ]; then
      echo "Component ${comp} disabled, continuing"
      break
    elif [ "${state}" == "Ready" ]; then
      echo "Component ${comp} ready, continuing"
      break
    fi
    if (( ${iter} >= 20 )); then
      # Wait for a total of 10 minutes for each component to complete
      echo "Timed out waiting for component ${comp} to reach Ready state"
      exit 1
    fi
    sleep 30s
  done
done
