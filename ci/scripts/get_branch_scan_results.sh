#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# NOTE: This script assumes that:
#
#   1) "docker login" has been done for the image registry
#   2) OCI credentials have been configured to allow the OCI CLI to fetch scan results from OCIR
#   3) "gh auth login" has been done to allow the github CLI to list releases and fetch release artifacts
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
RELEASE_SCRIPT_DIR=${SCRIPT_DIR}/../../release/scripts

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$CLEAN_BRANCH_NAME" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

# Hack to get the generated BOM from a release by pulling down the operator.yaml from the release artifacts
# and copying the BOM from the platform operator image
function get_bom_from_release() {
    local releaseTag=$1
    local outputFile=$2
    local tmpDir=$(mktemp -d)

    # Download the operator.yaml for the release and get the platform-operator image and tag
    gh release download ${releaseTag} -p 'operator.yaml' -D ${tmpDir}
    local image=$(grep "verrazzano-platform-operator:" ${tmpDir}/operator.yaml | grep "image:" -m 1 | xargs | cut -d' ' -f 2)

    # Create a container from the image and copy the BOM from the container
    local containerId=$(docker create ${image})
    docker cp ${containerId}:/verrazzano/platform-operator/verrazzano-bom.json ${outputFile}
    docker rm ${containerId}

    rm -fr ${tmpDir}
}

BOM_DIR=${WORKSPACE}/boms
mkdir -p ${BOM_DIR}
SCAN_RESULTS_BASE_DIR=${WORKSPACE}/scan-results
export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/latest
mkdir -p ${SCAN_RESULTS_DIR}

# Get the last pushed BOM for the tip of the branch
echo "Attempting to fetch BOM from object storage for branch: ${CLEAN_BRANCH_NAME}"
export SCAN_BOM_FILE=${BOM_DIR}/last-ocir-pushed-verrazzano-bom.json

oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/last-ocir-pushed-verrazzano-bom.json --file ${SCAN_BOM_FILE}
if [ $? -eq 0 ]; then
  echo "Fetching scan results for BOM: ${SCAN_BOM_FILE}"
  ${RELEASE_SCRIPT_DIR}/get_ocir_scan_results.sh
fi

if [[ "${CLEAN_BRANCH_NAME}" == release-* ]]; then
  # Get the list of matching releases, for example, on branch "release-1.0" the matching releases are "v1.0.0", "v1.0.1", ...
  echo "Attempting to fetch BOMs for released versions on branch: ${CLEAN_BRANCH_NAME}"

  MAJOR_MINOR_VERSION=${CLEAN_BRANCH_NAME:8}
  VERSIONS=$(gh release list | cut -f 3 | grep v${MAJOR_MINOR_VERSION})

  # For now get the results for all versions, at some point we should ignore versions that we no longer support
  for VERSION in ${VERSIONS}
  do
    echo "Fetching BOM for ${VERSION}"
    export SCAN_BOM_FILE=${BOM_DIR}/${VERSION}-bom.json
    get_bom_from_release ${VERSION} ${SCAN_BOM_FILE}

    export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/${VERSION}
    mkdir -p ${SCAN_RESULTS_DIR}

    echo "Fetching scan results for BOM: ${SCAN_BOM_FILE}"
    ${RELEASE_SCRIPT_DIR}/get_ocir_scan_results.sh
  done
fi
