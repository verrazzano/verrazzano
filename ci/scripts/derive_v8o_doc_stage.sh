#!/bin/bash

#
# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$VERRAZZANO_DEV_VERSION" ]; then
  echo "This script must only be called from Jenkins and requires environment variables VERRAZZANO_DEV_VERSION is set."
  exit 1
fi

docsVersion="v$(echo $VERRAZZANO_DEV_VERSION | cut -d '.' -f 1-2)"

# Phone-homing the URL with vz dev version to see if the docs exists
analysisUrl="https://verrazzano.io/$docsVersion/docs/troubleshooting/diagnostictools/analysisadvice/"

if curl --silent $analysisUrl | grep -q '404 Page not found' ; then
  docsVersion="devel"
fi
echo $docsVersion
