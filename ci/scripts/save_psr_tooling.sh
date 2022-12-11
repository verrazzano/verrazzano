#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -o pipefail
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

if [ -z "$JENKINS_URL" ] || [ -z "${PSR_PATH}" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cd ${PSR_PATH}/out
tar -czf ${WORKSPACE}/psrctl-linux-amd64.tar.gz -C linux_amd64 ./psrctl
tar -czf ${WORKSPACE}/psrctl-linux-arm64.tar.gz -C linux_arm64 ./psrctl
tar -czf ${WORKSPACE}/psrctl-darwin-amd64.tar.gz -C darwin_amd64 ./psrctl
tar -czf ${WORKSPACE}/psrctl-darwin-arm64.tar.gz -C darwin_arm64 ./psrctl

cd ${WORKSPACE}
sha256sum psrctl-linux-amd64.tar.gz > psrctl-linux-amd64.tar.gz.sha256
sha256sum psrctl-linux-arm64.tar.gz > psrctl-linux-arm64.tar.gz.sha256
sha256sum psrctl-darwin-amd64.tar.gz > psrctl-darwin-amd64.tar.gz.sha256
sha256sum psrctl-darwin-arm64.tar.gz > psrctl-darwin-arm64.tar.gz.sha256

# Push to ObjectStore
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-linux-amd64.tar.gz --file psrctl-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-linux-amd64.tar.gz.sha256 --file psrctl-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-linux-arm64.tar.gz --file psrctl-linux-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-linux-arm64.tar.gz.sha256 --file psrctl-linux-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-darwin-amd64.tar.gz --file psrctl-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-darwin-amd64.tar.gz.sha256 --file psrctl-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-darwin-arm64.tar.gz --file psrctl-darwin-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/psrctl-darwin-arm64.tar.gz.sha256 --file psrctl-darwin-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-linux-amd64.tar.gz --file psrctl-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-linux-amd64.tar.gz.sha256 --file psrctl-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-linux-arm64.tar.gz --file psrctl-linux-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-linux-arm64.tar.gz.sha256 --file psrctl-linux-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-darwin-amd64.tar.gz --file psrctl-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-darwin-amd64.tar.gz.sha256 --file psrctl-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-darwin-arm64.tar.gz --file psrctl-darwin-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/psrctl-darwin-arm64.tar.gz.sha256 --file psrctl-darwin-arm64.tar.gz.sha256
