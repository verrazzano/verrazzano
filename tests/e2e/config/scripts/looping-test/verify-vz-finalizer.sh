#!/bin/bash
#
# Copyright (C) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

finalizer=$(kubectl get vz -o=jsonpath='{.items[0].metadata.finalizers}')
echo "$finalizer"
if [[ -z "$finalizer" ]]; then
    echo "VZ CR is missing the finalizer necessary for uninstall"
    exit 1
fi