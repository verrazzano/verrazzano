#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Common script used to automate the release process.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z "$RELEASE_VERSION" ]; then
  echo "This script expects environment variable RELEASE_VERSION"
  exit 1
fi

# Release artifacts
declare -a releaseArtifacts=("verrazzano-platform-operator.yaml"
                             "verrazzano-platform-operator.yaml.sha256"
                             "verrazzano-${RELEASE_VERSION}-darwin-amd64.tar.gz"
                             "verrazzano-${RELEASE_VERSION}-darwin-amd64.tar.gz.sha256"
                             "verrazzano-${RELEASE_VERSION}-darwin-arm64.tar.gz"
                             "verrazzano-${RELEASE_VERSION}-darwin-arm64.tar.gz.sha256"
                             "verrazzano-${RELEASE_VERSION}-linux-amd64.tar.gz"
                             "verrazzano-${RELEASE_VERSION}-linux-amd64.tar.gz.sha256"
                             "verrazzano-${RELEASE_VERSION}-linux-arm64.tar.gz"
                             "verrazzano-${RELEASE_VERSION}-linux-arm64.tar.gz.sha256")
