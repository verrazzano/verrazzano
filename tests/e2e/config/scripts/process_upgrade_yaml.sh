#!/bin/bash

# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

function version_ge() { test "$(echo "$@" | tr " " "\n" | sort -rV | head -n 1)" == "$1"; }

# This script is used to add the version: field to the verrazzano custom resource .yaml file
# It is needed to test upgrade
VERSION=$1
CR_FILE=$2
VERSION_TO_USE=$VERSION

# remove leading v if it exists
if [[ "$VERSION_TO_USE" =~ ^v ]]; then
  VERSION_TO_USE=$(echo $VERSION_TO_USE | cut -c2-)
fi
echo Version without leading v is $VERSION_TO_USE

yq -i eval ".spec.version = \"v${VERSION_TO_USE}\"" ${CR_FILE}

if version_ge $VERSION_TO_USE "1.3.0"; then
  echo "$VERSION_TO_USE supports updates, testing update on upgrade scenario"
  # Add some simple additional updates to validate update during an upgrade
  yq -i eval '.spec.components.istio.ingress.kubernetes.replicas = 3' ${CR_FILE}
  yq -i eval '.spec.components.istio.egress.kubernetes.replicas = 3' ${CR_FILE}
fi
