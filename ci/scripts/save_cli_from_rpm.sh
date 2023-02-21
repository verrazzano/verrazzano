#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -e -o pipefail
set -xv

if [ -z "$1" ]; then
  echo "Branch name must be specified"
  exit 1
fi
CURRENT_BRANCH_NAME="$1"

if [ -z "$2" ]; then
  echo "Short commit hash must be specified"
  exit 1
fi
SHORT_COMMIT_HASH_ENV="$2"

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] ||
  [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$RPM_FILE" ] || [ -z "$TMP_BUILD_DIR" ] || [ -z "$BUILD_OS" ] ||
  [ -z "$BUILD_PLAT" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cp "${RPM_FILE}" "${TMP_BUILD_DIR}"
cd "${TMP_BUILD_DIR}"
rpm2cpio ./*64.rpm | cpio -idmv
mkdir -p "linux_${BUILD_PLAT}"
cp usr/bin/vz "linux_${BUILD_PLAT}"
tar -czf "${WORKSPACE}/vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz" -C "linux_${BUILD_PLAT}" .

cd "${WORKSPACE}"
sha256sum vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz >vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz --file vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256 --file vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz --file vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256 --file vz-linux-yum-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
