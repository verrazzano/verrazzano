#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z $1 ]; then
    echo "Kubernetes Version is required"
    exit 1
fi
KUBERNETES_VERSION=$1

# If the Kubernetes version starts with v, remove it
if [[ $KUBERNETES_VERSION == v* ]] ; then
  KUBERNETES_VERSION="${KUBERNETES_VERSION:1}"
fi

# If the Kubernetes version contains the patch version, remove it
COUNT_DOT=$(echo "$KUBERNETES_VERSION" | tr -cd "." | wc -c)

if [ "$COUNT_DOT" -gt "1" ]; then
  KUBERNETES_VERSION=$(echo ${KUBERNETES_VERSION} | cut -f1,2 -d'.')
fi

# Kubernetes version in the form <major version>.<minor version>
echo "$KUBERNETES_VERSION"