#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Waits until a namespace does not exist
#
NAMESPACE=$1
i=0
for (( i=1; i<=20; i++ ))
do
  kubectl get ns $NAMESPACE >&- 2>&-
  if [ "$?" -eq 1 ]; then
    exit 0
  fi
  echo "Waiting for namespace $NAMESPACE to be deleted ..."
  sleep 5
done
exit 1
