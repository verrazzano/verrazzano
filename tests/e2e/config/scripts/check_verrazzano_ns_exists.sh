#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

n=0
while [[ "$n" -lt 30 ]] && [[ $(kubectl get ns $1 -o 'jsonpath={..status.phase}') != "Active" ]]
do
   n=$(( n + 1 ))
   echo "waiting for namespace $1" && sleep 1
done

