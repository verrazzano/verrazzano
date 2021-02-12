#!/bin/bash

# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

#TYPES=$(kubectl api-resources --verbs=list -o name)
TYPES=`cat $SCRIPT_DIR/types.txt`

if [ -z $1 ] ; then
  echo "Please provide a namespace."
  exit 1
fi

for type in ${TYPES} ; do
  kubectl get "${type}" --show-kind -o custom-columns=NAME:.metadata.name,KIND:.kind --ignore-not-found --no-headers -n $1
done
