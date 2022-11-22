#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -xv

SECONDS=0
retval_success=1
retval_failed=1
i=0


resName=$(kubectl get vz -o jsonpath='{.items[*].metadata.name}')
echo "waiting for install of resource ${resName} to complete"

while [[ $retval_success -ne 0 ]] && [[ $retval_failed -ne 0 ]]  && [[ $i -lt 30 ]]  ; do
  sleep 60
  output=$(kubectl wait --for=condition=InstallFailed verrazzano/${resName} --timeout=0 2>&1)
  retval_failed=$?
  output=$(kubectl wait --for=condition=InstallComplete verrazzano/${resName} --timeout=0 2>&1)
  retval_success=$?
  i=$((i+1))
done

if [[ $retval_failed -eq 0 ]] || [[ $i -eq 30 ]] ; then
    echo "Installation Failed"
    kubectl get vz ${resName} -o yaml
    exit 1
fi

echo "Installation completed.  Wait time: $SECONDS seconds"