#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Updates release artifacts to object last-clean-periodic-test

if [ -z "$1" ]; then
  echo "The Verrazzano development version must be specified"
  exit 1
fi
DEVELOPENT_VERSION="$1"

if [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$CLEAN_BRANCH_NAME" ] || [ -z "$OCI_OS_REGION" ]; then
  echo "This script requires environment variables - CLEAN_BRANCH_NAME, OCI_OS_BUCKET, OCI_OS_NAMESPACE, OCI_OS_REGION and WORKSPACE"
  exit 1
fi

cd $WORKSPACE
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/operator.yaml --file operator.yaml
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-amd64.tar.gz --file vz-linux-amd64.tar.gz
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-amd64.tar.gz.sha256 --file vz-linux-amd64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-arm64.tar.gz --file vz-linux-arm64.tar.gz
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-linux-arm64.tar.gz.sha256 --file vz-linux-arm64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-amd64.tar.gz --file vz-darwin-amd64.tar.gz
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-amd64.tar.gz.sha256 --file vz-darwin-amd64.tar.gz.sha256
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-arm64.tar.gz --file vz-darwin-arm64.tar.gz
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/vz-darwin-arm64.tar.gz.sha256 --file vz-darwin-arm64.tar.gz.sha256

# Upload Verrazzano distributions
DISTRIBUTION_PREFIX="verrazzano-${VZ_DEVELOPENT_VERSION}"
VZ_LITE_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}-lite.zip"
VZ_LITE_RELEASE_BUNDLE_SHA256="${VZ_LITE_RELEASE_BUNDLE}.sha256"

VZ_FULL_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}.zip"
VZ_FULL_RELEASE_BUNDLE_SHA256="${VZ_FULL_RELEASE_BUNDLE}.sha256"

VZ_LITE_GENERATED="${WORKSPACE}/vz-lite-generated"
VZ_FULL_GENERATED="${WORKSPACE}/vz-full-generated"

oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_LITE_RELEASE_BUNDLE} --file ${VZ_LITE_GENERATED}/${VZ_LITE_RELEASE_BUNDLE}
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_LITE_RELEASE_BUNDLE_SHA256} --file ${VZ_LITE_GENERATED}/${VZ_LITE_RELEASE_BUNDLE_SHA256}

oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_FULL_RELEASE_BUNDLE} --file ${VZ_FULL_GENERATED}/${VZ_FULL_RELEASE_BUNDLE}
oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_FULL_RELEASE_BUNDLE_SHA256} --file ${VZ_FULL_GENERATED}/${VZ_FULL_RELEASE_BUNDLE_SHA256}

