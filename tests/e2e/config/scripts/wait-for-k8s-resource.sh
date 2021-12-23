#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

namespace=$1
condition=$2
resourceType=$3
all=${4:-false}
selector=$5

echo "Starting kubectl wait operation at $(date)"

selectorOption=""

if [ ! -z "$selector" ] ; then
  selectorOption="--selector=${selector}"
fi

allFlag=""

if [ $all == true ]; then
  allFlag="--all"
fi

retval=1
i=0

while [[ $retval -ne 0 ]] && [[ $i -lt 30 ]] ; do
  sleep 10
  output=$(kubectl wait --namespace ${namespace} --for=condition=$condition ${resourceType} ${selectorOption} ${allFlag} --timeout=0 2>&1)
  retval=$?
  i=$((i+1))
done

if [ $retval -eq 0 ] || [[ $i -eq 30 ]] ; then
    echo "Wait Failed"
    exit 1
fi

echo "Wait completed successfully at $(date)."
