#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Normally master and release-* branches are the only ones doing this, but there are other cases we also need to handle
#   1) we need to test out periodic pipeline changes
#   2) When new images are added to the BOM, folks need to be able to run registry tests and push to OCIR

# Exit when any command fails
set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
TOOL_SCRIPT_DIR=${SCRIPT_DIR}/../../tools/scripts
TEST_SCRIPT_DIR=${SCRIPT_DIR}/../../tests/e2e/config/scripts

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$OCIR_SCAN_REGISTRY" ] \
   || [ -z "$OCIR_SCAN_REPOSITORY_PATH" ] || [ -z "$OCIR_SCAN_COMPARTMENT" ] || [ -z "$OCIR_SCAN_TARGET" ] || [ -z "${CLEAN_BRANCH_NAME}" ] \
   || [ -z "$IS_PERIODIC_PIPELINE" ] || [ -z "$VERRAZZANO_IMAGES_DIRECTORY" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
else
  echo "INFO: push_to_ocir: basic environment provided"
fi

# We should have image tar files created already in $VERRAZZANO_IMAGES_DIRECTORY
if [ ! -d "${VERRAZZANO_IMAGES_DIRECTORY}" ]; then
  echo "No tar files were found to push into OCIR"
  exit 1
else
  echo "INFO: push_to_ocir: tar files found to push:"
  ls ${VERRAZZANO_IMAGES_DIRECTORY}
fi

BOM_FILE=${WORKSPACE}/tar-files/verrazzano-bom.json

if [ ! -f "${BOM_FILE}" ]; then
  echo "There is no verrazzano-bom.json from this run, so we can't push anything to OCIR"
  exit 1
else
  echo "INFO: push_to_ocir: BOM file found"
fi

# Periodic runs happen much more frequently than master promotions do, so we only conditionally do pushes to OCIR
# Note that not all runs that call this are periodic runs now.

# If we have a previous last-ocir-pushed-verrazzano-bom.json, then see if it matches the verrazzano-bom.json used
# to test with in this run. If they match, then we have already pushed the images for this verrazzano-bom.json
# into OCIR for this branches periodic runs and we do not need to do that again.
# If they don't match, or if we didn't have one to compare, then we will proceed to push them to OCIR
set +e
if [ -f "${WORKSPACE}/last-ocir-pushed-verrazzano-bom.json" ]; then
  diff ${WORKSPACE}/last-ocir-pushed-verrazzano-bom.json ${BOM_FILE} > /dev/null
  if [ $? -eq 0 ]; then
    echo "OCIR images for this verrazzano-bom.json have already been pushed to OCIR for scanning in a previous periodic run, skipping this step"
    exit 0
  else
    echo "INFO: push_to_ocir: previous BOM file found and had differences, proceeding to push "
  fi
else
  echo "INFO: push_to_ocir: no previous BOM file found to compare, proceeding to push"
fi
set -e

# This assumes that the docker login has happened, and that the OCI CLI has access as well with default profile

# We provide a single OCIR_SCAN_REPOSITORY_PATH as input, however the OCI CLI and the docker CLI requirements
# differ in terms of what needs to be included in the path. For the OCI CLI usages we need to trim the tenancy
# namespace from the path
TRIMMED_REPOSITORY_PATH=$(echo "$OCIR_SCAN_REPOSITORY_PATH" | cut -d / -f2-)

# We call the create repositories script, supplying the existing target information. If repositories are not
# targeted they will be created and targeted. If they are already targeted the script will skip trying to create them
# or updating the target. This is done to catch new images that get added in over time.
echo "INFO: push_to_ocir: call create_ocir_repositories"
sh $TEST_SCRIPT_DIR/create_ocir_repositories.sh -p $TRIMMED_REPOSITORY_PATH -r us-ashburn-1 -c $OCIR_SCAN_COMPARTMENT -t $OCIR_SCAN_TARGET -d ${VERRAZZANO_IMAGES_DIRECTORY}

# Push the images. NOTE: If a new image was added before we do the above "ensure" step, this may have the side
# effect of pushing that image to the root compartment rather than the desired sub-compartment (OCIR behaviour),
# and that new image will not be getting scanned until that is rectified (manually)
echo "INFO: push_to_ocir: call vz-registry-image-helper"
sh $TOOL_SCRIPT_DIR/vz-registry-image-helper.sh -t $OCIR_SCAN_REGISTRY -r $OCIR_SCAN_REPOSITORY_PATH -l ${VERRAZZANO_IMAGES_DIRECTORY} -b ${BOM_FILE}

# Finally push the current verrazzano-bom.json up as the last-ocir-pushed-verrazzano-bom.json so we know those were the latest images
# pushed up. This is used above for avoiding pushing things multiple times for no reason, and also is used when polling for
# results to know which images were last pushed (which results are the latest)

# NOTE: The normal workflow for master and release-* branches is NOT to do this. Those branches are getting OCIR pushes
# happening from the periodic tests normally. This is mainly to allow folks to push images from their branches to OCIR.
# So we need to understand if this is periodic or not, and also be careful to handle master/release branches accordingly here
echo "INFO: Pushing verrazzano-bom.json to object storage"
if [ "$IS_PERIODIC_PIPELINE" == "true" ]; then
  echo "INFO: Pushing verrazzano-bom.json to object storage for periodic pipeline. Scan results will show up under latest for ${CLEAN_BRANCH_NAME}"
  oci --auth instance_principal --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/last-ocir-pushed-verrazzano-bom.json --file ${BOM_FILE}
else
  if [[ "${CLEAN_BRANCH_NAME}" == "master" ]] || [[ "${CLEAN_BRANCH_NAME}" == release-* ]]; then
    echo "INFO: Pushing verrazzano-bom.json to object storage for non-periodic pipeline for master or release, Scan results are not normally tracked, these are stored under ${CLEAN_BRANCH_NAME}-last-snapshot/last-ocir-pushed-verrazzano-bom.json"
    oci --auth instance_principal --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-snapshot/last-ocir-pushed-verrazzano-bom.json --file ${BOM_FILE}
  else
    echo "INFO: Pushing verrazzano-bom.json to object storage for non-periodic pipeline, Scan results are NOT automatically tracked from this"
    oci --auth instance_principal --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/last-ocir-pushed-verrazzano-bom.json --file ${BOM_FILE}
  fi
fi
