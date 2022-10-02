#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Prepare the yaml for HA tests for the end-to-end test environment

set -o pipefail

set -xv

# Validate the install yaml file was passed and exists
if [ -z "$1" ]; then
  echo "Location of Verrazzano install file must be specified"
  exit 1
fi
INSTALL_CONFIG_FILE="$1"

if [ ! -f "$INSTALL_CONFIG_FILE" ]; then
  echo "The Verrazzano install file $INSTALL_CONFIG_FILE does not exist"
  exit 1
fi

# Update the install configuration to include what the end-to-end tests require
yq eval -i '.spec.components.opensearch.policies[0].policyName = "verrazzano-system"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[0].indexPattern = "verrazzano-system*"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[0].minIndexAge = "7d"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[0].rollover[0].minIndexAge = "1d"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[1].policyName = "verrazzano-application"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[1].indexPattern = "verrazzano-application*"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[1].minIndexAge = "7d"' "${INSTALL_CONFIG_FILE}"
yq eval -i '.spec.components.opensearch.policies[1].rollover[0].minIndexAge = "1d"' "${INSTALL_CONFIG_FILE}"

exit 0
