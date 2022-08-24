#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Normally master and release-* branches are the only ones doing this, but when we need to test out pipeline changes we can make use
# as well

if [ -z "$1" ]; then
  echo "GIT commit must be specified"
  exit 1
fi
GIT_COMMIT_USED="$1"

if [ -z "$2" ]; then
  echo "Short commit hash must be specified"
  exit 1
fi
SHORT_COMMIT_HASH_ENV="$2"

if [ -z "$3" ]; then
  echo "The tar/Zip file prefix must be specified"
  exit 1
fi
ZIPFILE_PREFIX="$3"

if [ -z "$4" ]; then
  echo "The Verrazzano development version must be specified"
  exit 1
fi
DEVELOPENT_VERSION="$4"

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$CLEAN_BRANCH_NAME" ] || [ -z "$BRANCH_NAME" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

# We originally handled only "master" here. "master" and "release-*" branches do not have "/" in them, however we do have
# those in feature branches. This causes problems in some situations, so we have 2 variants for the branch names being used here:
#      BRANCH_NAME may be a path with /
#      CLEAN_BRANCH_NAME has the / replaced with %2F so it is not treated as a path
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/operator.yaml --file operator.yaml
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/operator.yaml --file operator.yaml
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz --file vz-linux-amd64.tar.gz
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz.sha256 --file vz-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz --file vz-linux-arm64.tar.gz
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz.sha256 --file vz-linux-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz --file vz-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz.sha256 --file vz-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz --file vz-darwin-arm64.tar.gz
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz.sha256 --file vz-darwin-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-amd64.tar.gz --file vz-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-amd64.tar.gz.sha256 --file vz-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-arm64.tar.gz --file vz-linux-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-arm64.tar.gz.sha256 --file vz-linux-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-amd64.tar.gz --file vz-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-amd64.tar.gz.sha256 --file vz-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-arm64.tar.gz --file vz-darwin-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-arm64.tar.gz.sha256 --file vz-darwin-arm64.tar.gz.sha256

# Generate a Verrazzano full Zip for private registry testing

# Get the latest stable generated BOM file
local_bom=${WORKSPACE}/verrazzano-bom.json
last_ocir_pushed_bom=${WORKSPACE}/last-ocir-pushed-verrazzano-bom.json
mkdir -p $(dirname ${local_bom}) || true
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/generated-verrazzano-bom.json --file ${local_bom}
# NOTE: The first time we run through for a branch we do not have a last-ocir-pushed-verrazzano-bom.bom present yet in object storage (there is no previous run), so we ignore if it fails to find one here
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/last-ocir-pushed-verrazzano-bom.json --file ${last_ocir_pushed_bom} || true
# Call the script to generate and publish the BOM
echo "Creating Zip for commit ${GIT_COMMIT_USED}, short hash ${SHORT_COMMIT_HASH_ENV}, file prefix ${ZIPFILE_PREFIX}, BOM file ${local_bom}"
ci/scripts/generate_product_zip.sh ${GIT_COMMIT_USED} ${SHORT_COMMIT_HASH_ENV} ${CLEAN_BRANCH_NAME}-last-clean-periodic-test ${ZIPFILE_PREFIX} ${local_bom}

# Note: We have Verrazzano images tar files locally under ${WORKSPACE}/tar-files
# Move them to a new directory (rather than changing the vz-registry-image-helper.sh) and use the new directory from here onwards in the periodic job
# The new directory is defined in the Jenkins script as an environment variable VERRAZZANO_IMAGES_DIRECTORY
#
echo "Creating Verrazzano Release Distribution bundles"
cd ${WORKSPACE}
ci/scripts/generate_vz_distribution.sh ${WORKSPACE} ${local_bom} ${DEVELOPENT_VERSION}
