#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

INVALID_OPERATOR_YAML=$1
VPO_YAML=$2
VALIDATION_ERROR="ValidationError"

cd "$WORKSPACE"
ERROR1=$(./vz install --filename "$INVALID_OPERATOR_YAML" --manifests "$VPO_YAML" 2>&1 >/dev/null)  
if [[ "$ERROR1" =~ .*"$VALIDATION_ERROR".* ]]; then
    echo "Expected Error: $VALIDATION_ERROR, Actual Error: $VALIDATION_ERROR"
    echo "Error: $VALIDATION_ERROR was caught"
else 
    echo "Expected Error: $VALIDATION_ERROR from $INVALID_OPERATOR_YAML, Actual Error: $ERROR1" 
    exit 1
fi

ERROR2=$(./vz install --set trash=foo --manifests "$VPO_YAML" 2>&1 >/dev/null)
if [[ "$ERROR2" =~ .*"$VALIDATION_ERROR".* ]]; then
    echo "Expected Error: $VALIDATION_ERROR, Actual Error: $VALIDATION_ERROR"
    echo "Error: $VALIDATION_ERROR was caught"
else
    echo "Expected Error: $VALIDATION_ERROR from field(s) trash=foo, Actual Error: $ERROR2"
    exit 1
fi