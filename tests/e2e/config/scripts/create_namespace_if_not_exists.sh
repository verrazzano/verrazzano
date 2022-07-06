#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

NAMESPACE=$1

# Create a namespace if it does not already exist
if [ "$(kubectl get ns "${NAMESPACE}" -o 'jsonpath={..status.phase}')" != "Active" ]
then
  kubectl create namespace "${NAMESPACE}"
fi
