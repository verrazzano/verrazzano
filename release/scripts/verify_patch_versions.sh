#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

EXPECTED_VERSION=${1}

# get the version from the development version file
VERRAZZANO_DEV_VERSION=$(grep verrazzano-development-version ${SCRIPT_DIR}/../../.verrazzano-development-version | sed -e 's/verrazzano-development-version=//')

if [ "${VERRAZZANO_DEV_VERSION}" != "${EXPECTED_VERSION}" ]; then
  echo "Wrong version found in development version file"
  exit 1
fi

# get the version from the VZ chart
VERRAZZANO_ROOT_DIR=$(realpath ${SCRIPT_DIR}/../..)
NUM_CHARTS_WITH_VERSION=$(grep -r  --include "Chart.yaml" "version: ${1}" ${VERRAZZANO_ROOT_DIR}/platform-operator/helm_config/charts | cut -d: -f3- |\cut -d ' ' -f4 | sort | uniq -c)
echo "$NUM_CHARTS_WITH_VERSION"
if [ $NUM_CHARTS_WITH_VERSION -ne 3 ]; then
  echo "One of the Verrazzano yaml charts has a version value that differs from the expected version of ${EXPECTED_VERSION}"
  exit 1
fi


