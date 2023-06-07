#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

OPERATOR_YAML=$1
VPO_YAML=$2
VALIDATION_ERROR="ValidationError"
cd "$WORKSPACE"

ERROR1=$(./vz install --filename "$OPERATOR_YAML" --manifests "$VPO_YAML" 2>&1 >/dev/null)  
if [[ "$ERROR1" =~ .*"$VALIDATION_ERROR".* ]]; then
    echo "Error: $VALIDATION_ERROR was caught as expected"
else 
    echo "Expected $VALIDATION_ERROR in invalidCR.yaml" 
    exit 1
fi

ERROR2=$(./vz install --set trash=foo --manifests "$VPO_YAML" 2>&1 >/dev/null)
if [[ "$ERR2" =~ .*"$VALIDATION_ERROR".* ]]; then
    echo "Error was caught as expected"
else
    echo "Expected $VALIDATION_ERROR from field(s) trash=foo"
    exit 1
fi