#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -x
NAMESPACES=${1}
for ns in ${NAMESPACES[@]}
do
  kubectl get pods -n ${ns} -o jsonpath='{range .items[*]}{.metadata.name}{" "}{..uid}{"\n"}{end}'
done
