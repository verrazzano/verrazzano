#!/usr/bin/env bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SECONDS=0
retval_success=1
retval_failed=1
i=0

resName=$(kubectl get vz -o jsonpath='{.items[*].metadata.name}')
echo "waiting for install of resource ${resName} to complete"

sleep 10
while [[ $i -lt 30 ]]; do
  output=$(kubectl wait --for=condition=InstallFailed verrazzano/${resName} --timeout=0 2>&1)
  retval_failed=$?
  output=$(kubectl wait --for=condition=InstallComplete verrazzano/${resName} --timeout=0 2>&1)
  retval_success=$?
  if [[ $retval_success -ne 0 ]] || [[ $retval_failed -ne 0 ]]; then
    break
  fi
  i=$((i+1))
  sleep 60
done

if [[ $retval_failed -eq 0 ]] || [[ $i -eq 30 ]] ; then
    echo "Installation Failed"
    kubectl get vz my-verrazzano -o yaml
    exit 1
fi

echo "Installation completed.  Wait time: $SECONDS seconds"