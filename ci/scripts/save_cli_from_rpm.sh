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
  [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$BUILD_OS" ] || [ -z "$BUILD_PLAT" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

RPM_FILE=$(find "${MODULE_REPO_ARCHIVE_DIR}"/results -name \*64.rpm)
cp "${RPM_FILE}" "${WORKSPACE}"
cd "${WORKSPACE}"
rpm2cpio ./*64.rpm | cpio -idmv
mkdir -p "linux_${BUILD_PLAT}"
cp usr/bin/vz "linux_${BUILD_PLAT}"
tar -czf "${WORKSPACE}/vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz" -C "linux_${BUILD_PLAT}" .

cd "${WORKSPACE}"
sha256sum vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz >vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz --file vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256 --file vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz --file vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256 --file vz-cli-from-rpm-linux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256

# Save generated module stream repo
tar -czf ${WORKSPACE}/vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz -C ${WORKSPACE}/modulerepo .
cd ${WORKSPACE}
sha256sum vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz >vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz --file vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256 --file vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH}/vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz --file vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH}/vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256 --file vz-cli-yum-repo-llinux-${BUILD_OS}-${BUILD_PLAT}.tar.gz.sha256
