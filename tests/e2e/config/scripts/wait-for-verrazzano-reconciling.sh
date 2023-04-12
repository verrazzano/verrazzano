#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

i=0

resName=$(kubectl get vz -o jsonpath='{.items[*].metadata.name}')
echo "waiting for resource ${resName} to be reconciling"
while [[ $i -lt 45 ]]  ; do
  sleep 60
  vzstate=$(kubectl get vz ${resName} -o jsonpath={.status.state})
  echo "vz/${resName} state: ${vzstate}"
  if [ "${vzstate}" == "Reconciling" ]; then
    exit 0
  fi
  i=$((i+1))
done

kubectl get vz ${resName}
