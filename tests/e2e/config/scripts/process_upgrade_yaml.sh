#!/bin/bash

# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

function version_ge() { test "$(echo "$@" | tr " " "\n" | sort -rV | head -n 1)" == "$1"; }

# This script is used to add the version: field to the verrazzano custom resource .yaml file
# It is needed to test upgrade
VERSION=$1
CR_FILE=$2
yq -i eval ".spec.version = \"v${VERSION}\"" ${CR_FILE}

if version_ge $VERSION "1.3.0"; then
  if [ "$CRD_API_VERSION" == "v1alpha1" ]; then
    echo "$VERSION supports updates, testing update on upgrade scenario"
    # Add some simple additional updates to validate update during an upgrade
    yq -i eval '.spec.components.istio.ingress.kubernetes.replicas = 3' ${CR_FILE}
    yq -i eval '.spec.components.istio.egress.kubernetes.replicas = 3' ${CR_FILE}
  elif [ "$CRD_API_VERSION" == "v1beta1" ]; then
    yq -i eval '.spec.components.istio.overrides.[0].values.ingressNGINX.spec.ingressGateways.k8s = 3' ${CR_FILE}
    yq -i eval '.spec.components.istio.overrides.[0].values.egressga.replicas = 3' ${CR_FILE}
  fi
fi
