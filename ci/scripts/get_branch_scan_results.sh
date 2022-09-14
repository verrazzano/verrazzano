#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$OCI_SCAN_BUCKET" ] || [ -z "$CLEAN_BRANCH_NAME" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

BASE_OBJ_PATH="daily-scan/${CLEAN_BRANCH_NAME}"
SCAN_DATETIME="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
JOB_OBJ_PATH="${BASE_OBJ_PATH}/${SCAN_DATETIME}-${BUILD_NUMBER}"

# Hack to get the generated BOM from a release by pulling down the operator.yaml from the release artifacts
# and copying the BOM from the platform operator image
function get_bom_from_release() {
    local releaseTag=$1
    local outputFile=$2
    local tmpDir=$(mktemp -d)

    # Download the operator.yaml for the release and get the platform-operator image and tag
    local operator_yaml=$(derive_platform_operator "$releaseTag")
    gh release download ${releaseTag} -p '${operator_yaml}' -D ${tmpDir}
    local image=$(grep "verrazzano-platform-operator:" ${tmpDir}/${operator_yaml} | grep "image:" -m 1 | xargs | cut -d' ' -f 2)

    # Create a container from the image and copy the BOM from the container
    local containerId=$(docker create ${image})
    docker cp ${containerId}:/verrazzano/platform-operator/verrazzano-bom.json ${outputFile}
    docker rm ${containerId}

    rm -fr ${tmpDir}
}

# Publish the results to object storage
function publish_results() {
    local resultName=$1
    local bomFile=$2
    local resultsDir=$3

    zip -r ${WORKSPACE}/${resultName}-details.zip ${resultsDir}

    # Push latest
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${BASE_OBJ_PATH}/${resultName}/verrazzano-bom.json --file ${bomFile}
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${BASE_OBJ_PATH}/${resultName}/consolidated-report.out --file ${resultsDir}/consolidated-report.out
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${BASE_OBJ_PATH}/${resultName}/consolidated.csv --file ${resultsDir}/consolidated.csv
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${BASE_OBJ_PATH}/${resultName}/consolidated-upload.json --file ${resultsDir}/consolidated-upload.json
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${BASE_OBJ_PATH}/${resultName}/details.zip --file ${WORKSPACE}/${resultName}-details.zip

    # Push to job specific location
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${JOB_OBJ_PATH}/${resultName}/verrazzano-bom.json --file ${bomFile}
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${JOB_OBJ_PATH}/${resultName}/consolidated-report.out --file ${resultsDir}/consolidated-report.out
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${JOB_OBJ_PATH}/${resultName}/consolidated.csv --file ${resultsDir}/consolidated.csv
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${JOB_OBJ_PATH}/${resultName}/consolidated-upload.json --file ${resultsDir}/consolidated-upload.json
    OCI_CLI_AUTH="instance_principal" oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_SCAN_BUCKET} --name ${JOB_OBJ_PATH}/${resultName}/details.zip --file ${WORKSPACE}/${resultName}-details.zip
}

# Read the value for a given key from effective.config.json
function derive_platform_operator() {
  local release_tag="$1"

  # Remove prefix v from version
  local version_num=${release_tag:1}

  # Verrazzano distribution from 1.4.0 release replaces operator.yaml with verrazzano-platform-operator.yaml in release assets
  local version_14=1.4.0
  local operator_yaml=$(echo ${VERSION_NUM} ${VERSION_14} | awk '{if ($1 < $2) print "operator.yaml"; else print "verrazzano-platform-operator.yaml"}')
  echo $operator_yaml
  return 0
}

BOM_DIR=${WORKSPACE}/boms
mkdir -p ${BOM_DIR}
SCAN_RESULTS_BASE_DIR=${WORKSPACE}/scan-results
export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/latest
mkdir -p ${SCAN_RESULTS_DIR}

# Where the results are kept for the branch depend on what kind of branch it is and where the updated bom is stored:
#    master, release-* branches are regularly updated using the periodic pipelines only
#
#        The BOM for the latest results from the NORMAL workflows is here (master, release-*, special runs of branches):
#             ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/last-ocir-pushed-verrazzano-bom.json
#
#        It is possible that someone ran a job which needed to specify that the tip of master or release-* push images to
#        OCIR. This does NOT happen normally, the only situation where this is done from a pipeline is when performing a
#        release that required a BUILD to be done (ie: when releasing something that was NOT pre-baked for some reason).
#        In these cases, the BOM is stored here:
#
#             ${CLEAN_BRANCH_NAME}-last-snapshot/last-ocir-pushed-verrazzano-bom.json
#
#    all other branches only will be pushed if explicitly set as a parameter. In these cases, the BOM is stored here:
#
#             ${CLEAN_BRANCH_NAME}/last-ocir-pushed-verrazzano-bom.json

# Get the last pushed BOMs for the branch
echo "Attempting to fetch BOM from object storage for branch: ${CLEAN_BRANCH_NAME}"
mkdir -p ${BOM_DIR}/${CLEAN_BRANCH_NAME}-last-clean-periodic-test
mkdir -p ${BOM_DIR}/${CLEAN_BRANCH_NAME}-last-snapshot
mkdir -p ${BOM_DIR}/${CLEAN_BRANCH_NAME}
export SCAN_BOM_PERIODIC_PATH=${CLEAN_BRANCH_NAME}-last-clean-periodic-test/last-ocir-pushed-verrazzano-bom.json
export SCAN_BOM_SNAPSHOT_PATH=${CLEAN_BRANCH_NAME}-last-snapshot/last-ocir-pushed-verrazzano-bom.json
export SCAN_BOM_FEATURE_PATH=${CLEAN_BRANCH_NAME}/last-ocir-pushed-verrazzano-bom.json
export SCAN_LAST_PERIODIC_BOM_FILE=${BOM_DIR}/${SCAN_BOM_PERIODIC_PATH}
export SCAN_LAST_SNAPSHOT_BOM_FILE=${BOM_DIR}/${SCAN_BOM_SNAPSHOT_PATH}
export SCAN_FEATURE_BOM_FILE=${BOM_DIR}/${SCAN_BOM_FEATURE_PATH}
export SCAN_COMMIT_PERIODIC_PATH=${CLEAN_BRANCH_NAME}-last-clean-periodic-test/verrazzano_periodic-commit.txt
export SCAN_LAST_PERIODIC_COMMIT_FILE=${BOM_DIR}/${SCAN_COMMIT_PERIODIC_PATH}

# If there is a periodic BOM file for this branch, get those results
GIT_COMMIT="TBD-Commit"
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SCAN_BOM_PERIODIC_PATH} --file ${SCAN_LAST_PERIODIC_BOM_FILE} 2> /dev/null
if [ $? -eq 0 ]; then
  echo "Fetching scan results for BOM: ${SCAN_LAST_PERIODIC_BOM_FILE}"
  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SCAN_COMMIT_PERIODIC_PATH} --file ${SCAN_LAST_PERIODIC_COMMIT_FILE} 2> /dev/null
  if [ $? -eq 0 ]; then
    GIT_COMMIT=$(cat ${SCAN_LAST_PERIODIC_COMMIT_FILE} | cut -d'=' -f2)
  fi
  export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/latest-periodic
  mkdir -p ${SCAN_RESULTS_DIR}
  ${RELEASE_SCRIPT_DIR}/scan_bom_images.sh  -b ${SCAN_LAST_PERIODIC_BOM_FILE} -o ${SCAN_RESULTS_DIR} -r ${OCIR_SCAN_REGISTRY} -x ${OCIR_REPOSITORY_BASE}
  ${RELEASE_SCRIPT_DIR}/get_ocir_scan_results.sh ${SCAN_LAST_PERIODIC_BOM_FILE}
  ${RELEASE_SCRIPT_DIR}/generate_vulnerability_report.sh ${SCAN_RESULTS_DIR} ${GIT_COMMIT} ${CLEAN_BRANCH_NAME} "periodic" ${SCAN_DATETIME} ${BUILD_NUMBER}
  ${RELEASE_SCRIPT_DIR}/generate_upload_file.sh ${SCAN_RESULTS_DIR}/consolidated.csv "periodic" > ${SCAN_RESULTS_DIR}/consolidated-upload.json
  publish_results "last-clean-periodic-test" ${SCAN_LAST_PERIODIC_BOM_FILE} ${SCAN_RESULTS_DIR}
else
  echo "INFO: Did not find a periodic BOM for ${CLEAN_BRANCH_NAME}"
  rm ${SCAN_LAST_PERIODIC_BOM_FILE} || true
fi

# If there is a snapshot BOM file for this branch, get those results
GIT_COMMIT="TBD-Commit"
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SCAN_BOM_SNAPSHOT_PATH} --file ${SCAN_LAST_SNAPSHOT_BOM_FILE} 2> /dev/null
if [ $? -eq 0 ]; then
  echo "Fetching scan results for BOM: ${SCAN_LAST_SNAPSHOT_BOM_FILE}"
  export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/last-snapshot-possibly-old
  mkdir -p ${SCAN_RESULTS_DIR}
  ${RELEASE_SCRIPT_DIR}/scan_bom_images.sh  -b ${SCAN_LAST_SNAPSHOT_BOM_FILE} -o ${SCAN_RESULTS_DIR} -r ${OCIR_SCAN_REGISTRY} -x ${OCIR_REPOSITORY_BASE}
  ${RELEASE_SCRIPT_DIR}/get_ocir_scan_results.sh ${SCAN_LAST_SNAPSHOT_BOM_FILE}
  ${RELEASE_SCRIPT_DIR}/generate_vulnerability_report.sh ${SCAN_RESULTS_DIR} ${GIT_COMMIT} ${CLEAN_BRANCH_NAME} "snapshot" ${SCAN_DATETIME} ${BUILD_NUMBER}
  ${RELEASE_SCRIPT_DIR}/generate_upload_file.sh ${SCAN_RESULTS_DIR}/consolidated.csv "snapshot" > ${SCAN_RESULTS_DIR}/consolidated-upload.json
  publish_results "last-snapshot" ${SCAN_LAST_SNAPSHOT_BOM_FILE} ${SCAN_RESULTS_DIR}
else
  echo "INFO: Did not find a snapshot BOM for ${CLEAN_BRANCH_NAME}"
  rm ${SCAN_LAST_SNAPSHOT_BOM_FILE} || true
fi

# If this is a feature branch, get those results
GIT_COMMIT="TBD-Commit"
if [[ "${CLEAN_BRANCH_NAME}" != "master" ]] && [[ "${CLEAN_BRANCH_NAME}" != release-* ]]; then
  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SCAN_BOM_FEATURE_PATH} --file ${SCAN_FEATURE_BOM_FILE} 2> /dev/null
  if [ $? -eq 0 ]; then
    echo "Fetching scan results for BOM: ${SCAN_FEATURE_BOM_FILE}"
    export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/feature-branch-latest
    mkdir -p ${SCAN_RESULTS_DIR}
    ${RELEASE_SCRIPT_DIR}/scan_bom_images.sh  -b ${SCAN_FEATURE_BOM_FILE} -o ${SCAN_RESULTS_DIR} -r ${OCIR_SCAN_REGISTRY} -x ${OCIR_REPOSITORY_BASE}
    ${RELEASE_SCRIPT_DIR}/get_ocir_scan_results.sh ${SCAN_FEATURE_BOM_FILE}
    ${RELEASE_SCRIPT_DIR}/generate_vulnerability_report.sh ${SCAN_RESULTS_DIR} ${GIT_COMMIT} ${CLEAN_BRANCH_NAME} "feature" ${SCAN_DATETIME} ${BUILD_NUMBER}
    ${RELEASE_SCRIPT_DIR}/generate_upload_file.sh ${SCAN_RESULTS_DIR}/consolidated.csv "feature" > ${SCAN_RESULTS_DIR}/consolidated-upload.json
    publish_results "feature" ${SCAN_FEATURE_BOM_FILE} ${SCAN_RESULTS_DIR}
  else
    echo "INFO: Did not find a feature BOM for ${CLEAN_BRANCH_NAME}"
    rm ${SCAN_FEATURE_BOM_FILE} || true
  fi
fi

if [[ "${CLEAN_BRANCH_NAME}" == release-* ]]; then
  # Get the list of matching releases, for example, on branch "release-1.0" the matching releases are "v1.0.0", "v1.0.1", ...
  echo "Attempting to fetch BOMs for released versions on branch: ${CLEAN_BRANCH_NAME}"

  MAJOR_MINOR_VERSION=${CLEAN_BRANCH_NAME:8}
  VERSIONS=$(gh release list | cut -f 3 | grep v${MAJOR_MINOR_VERSION})

  # For now get the results for all versions, at some point we should ignore versions that we no longer support
  for VERSION in ${VERSIONS}
  do
    GIT_COMMIT=$(git rev-list -n 1 ${VERSION})
    echo "Fetching BOM for ${VERSION}"
    export SCAN_BOM_FILE=${BOM_DIR}/${VERSION}-bom.json
    get_bom_from_release ${VERSION} ${SCAN_BOM_FILE}

    export SCAN_RESULTS_DIR=${SCAN_RESULTS_BASE_DIR}/${VERSION}
    mkdir -p ${SCAN_RESULTS_DIR}

    echo "Fetching scan results for BOM: ${SCAN_BOM_FILE}"
    ${RELEASE_SCRIPT_DIR}/scan_bom_images.sh  -b ${SCAN_BOM_FILE} -o ${SCAN_RESULTS_DIR} -r ${OCIR_SCAN_REGISTRY} -x ${OCIR_REPOSITORY_BASE}
    ${RELEASE_SCRIPT_DIR}/get_ocir_scan_results.sh ${SCAN_BOM_FILE}
    ${RELEASE_SCRIPT_DIR}/generate_vulnerability_report.sh ${SCAN_RESULTS_DIR} ${GIT_COMMIT} ${CLEAN_BRANCH_NAME} ${VERSION} ${SCAN_DATETIME} ${BUILD_NUMBER}
    ${RELEASE_SCRIPT_DIR}/generate_upload_file.sh ${SCAN_RESULTS_DIR}/consolidated.csv "${VERSION}" > ${SCAN_RESULTS_DIR}/consolidated-upload.json
    publish_results ${VERSION} ${SCAN_BOM_FILE} ${SCAN_RESULTS_DIR}
  done
fi
