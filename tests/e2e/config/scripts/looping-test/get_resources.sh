#!/bin/bash

# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

TYPES=`cat $SCRIPT_DIR/types.txt`
INCLUDE_CRDS=true

if [ -z "$1" ] ; then
  echo "Please provide a namespace."
  exit 1
fi

if [ -n "$2" ] ; then
  INCLUDE_CRDS=$2
fi

namespace=$1
for type in ${TYPES} ; do
  if [ "$type" == "CustomResourceDefinition"  ] || [ "$type" == "APIService" ] || [ "$type" == "ValidatingWebhookConfiguration" ]; then
    if [ "${INCLUDE_CRDS}" != true ]; then
      continue
    fi
  fi

  echo "kubectl get ${type} --show-kind -o custom-columns=NAME:.metadata.name,KIND:.kind --ignore-not-found --no-headers -n ${namespace}"
  kubectl get "${type}" --show-kind -o custom-columns=NAME:.metadata.name,KIND:.kind --ignore-not-found --no-headers -n ${namespace}
  echo
done
