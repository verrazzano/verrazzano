#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

OPERATOR_YAML=$1
VPO_YAML=$2
cd "$WORKSPACE"
ERROR1=$(./vz install --filename "$OPERATOR_YAML" --manifests "$VPO_YAML" 2>&1 >/dev/null)
ERROR2=$(./vz install --set trash=foo --manifests "$VPO_YAML" 2>&1 >/dev/null)
 
ERR1=$(echo "$ERROR1" | grep ValidationError)
if [[ "$ERR1" -eq 0 ]]; then
    exit 0
else 
    echo "Expected ValidationError in invalidCR.yaml" 
    exit 1
fi

ERR2=$(echo "$ERROR2" | grep ValidationError)
if [[ "$ERR2" -eq 0 ]]; then
    exit 0
else
    echo "Expected ValidationError from field(s) trash=foo"
    exit 1
fi