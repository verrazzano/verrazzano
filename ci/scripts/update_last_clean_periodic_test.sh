#!/usr/bin/env bash
#
# Copyright (c) 2022, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Updates release artifacts to object last-clean-periodic-test

set -e

if [ -z "$1" ]; then
  echo "The Verrazzano development version must be specified"
  exit 1
fi
DEVELOPMENT_VERSION="$1"

if [ -z "$2" ]; then
  echo "Short commit hash must be specified"
  exit 1
fi
SHORT_COMMIT_HASH_ENV="$2"

# Parameter 3 is optional, when specified it indicates that we are FORCING a specific commit in as the last clean periodic test
# effectively making it useable as a release candidate
# This ONLY is ever specified when we are FORCING a specific periodic run in so that we can use it for a release. There are specific
# checks that are done before this is called, so when we get here we are just moving things to the expected places
# This WILL reset the last clean commit though to something that may not really have been fully clean in a periodic run. If we are
# doing a FORCE we have already explicitly determined that is OK (either we replayed the specific runs that had failed, or we determined
# the failed result was OK otherwise in some way)
if [ ! -z "$3" ]; then
  echo "FORCING ${GIT_COMMIT_USED} to be the last clean periodic commit"
fi

if [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$CLEAN_BRANCH_NAME" ] || [ -z "$BRANCH_NAME" ] || [ -z "$OCI_OS_REGION" ] || [ -z "$GIT_COMMIT_USED" ]; then
  echo "This script requires environment variables - CLEAN_BRANCH_NAME, OCI_OS_BUCKET, OCI_OS_COMMIT_BUCKET, OCI_OS_NAMESPACE, OCI_OS_REGION, GIT_COMMIT_USED, and WORKSPACE"
  exit 1
fi

cd $WORKSPACE

DISTRIBUTION_PREFIX="verrazzano-${DEVELOPMENT_VERSION}"
VZ_LITE_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}-lite.zip"
VZ_LITE_RELEASE_BUNDLE_SHA256="${VZ_LITE_RELEASE_BUNDLE}.sha256"

VZ_FULL_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}.zip"
VZ_FULL_RELEASE_BUNDLE_SHA256="${VZ_FULL_RELEASE_BUNDLE}.sha256"

# First confirm that the files we need to copy into place actually exist at the source location (fail first if they aren't all there). This is mainly
# for when we are being asked to FORCE an existing specific commit in as the last clean periodic test to sanity check it, but it doesn't hurt to always
# confirm everything really is there and explicitly fail if not.
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/verrazzano_${DEVELOPMENT_VERSION}-images.txt
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/operator.yaml
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/generated-verrazzano-bom.json
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_LITE_RELEASE_BUNDLE}
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_LITE_RELEASE_BUNDLE_SHA256}
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_FULL_RELEASE_BUNDLE}
oci --region ${OCI_OS_REGION} os object head --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_FULL_RELEASE_BUNDLE_SHA256}

if [ "$3" == "DRY_RUN" ]; then
   echo "Only doing a DRY_RUN, validated that the artifacts for ${GIT_COMMIT_USED} exist and are able to be used for a candidate"
   exit 0
fi

# Update the clean periodic commit
echo "git-commit=${GIT_COMMIT_USED}" > commit-that-passed.txt
cat commit-that-passed.txt
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/verrazzano_periodic-commit.txt --file commit-that-passed.txt

# Update the artifacts
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/verrazzano_${DEVELOPMENT_VERSION}-images.txt --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/verrazzano_${DEVELOPMENT_VERSION}-images.txt
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/operator.yaml --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/operator.yaml
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/generated-verrazzano-bom.json --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/generated-verrazzano-bom.json
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-amd64.tar.gz
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz.sha256 --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-amd64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-arm64.tar.gz
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz.sha256 --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-arm64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-amd64.tar.gz
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz.sha256 --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-amd64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-arm64.tar.gz
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz.sha256 --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-arm64.tar.gz.sha256

# Update Verrazzano distributions
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_LITE_RELEASE_BUNDLE} --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_LITE_RELEASE_BUNDLE}
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_LITE_RELEASE_BUNDLE_SHA256} --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_LITE_RELEASE_BUNDLE_SHA256}

oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_FULL_RELEASE_BUNDLE} --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_FULL_RELEASE_BUNDLE}
oci --region ${OCI_OS_REGION} os object copy --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --destination-bucket ${OCI_OS_BUCKET} --source-object-name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_FULL_RELEASE_BUNDLE_SHA256} --destination-object-name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_FULL_RELEASE_BUNDLE_SHA256}
