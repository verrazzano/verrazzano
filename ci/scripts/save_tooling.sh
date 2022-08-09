#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

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

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cd out
tar -czf ${WORKSPACE}/vz-linux-amd64.tar.gz -C linux_amd64 .
tar -czf ${WORKSPACE}/vz-linux-arm64.tar.gz -C linux_arm64 .
tar -czf ${WORKSPACE}/vz-darwin-amd64.tar.gz -C darwin_amd64 .
tar -czf ${WORKSPACE}/vz-darwin-arm64.tar.gz -C darwin_arm64 .

cd ${WORKSPACE}
sha256sum vz-linux-amd64.tar.gz > vz-linux-amd64.tar.gz.sha256
sha256sum vz-linux-arm64.tar.gz > vz-linux-arm64.tar.gz.sha256
sha256sum vz-darwin-amd64.tar.gz > vz-darwin-amd64.tar.gz.sha256
sha256sum vz-darwin-arm64.tar.gz > vz-darwin-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-linux-amd64.tar.gz --file vz-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-linux-amd64.tar.gz.sha256 --file vz-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-linux-arm64.tar.gz --file vz-linux-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-linux-arm64.tar.gz.sha256 --file vz-linux-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-darwin-amd64.tar.gz --file vz-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-darwin-amd64.tar.gz.sha256 --file vz-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-darwin-arm64.tar.gz --file vz-darwin-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/vz-darwin-arm64.tar.gz.sha256 --file vz-darwin-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz --file vz-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-amd64.tar.gz.sha256 --file vz-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz --file vz-linux-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-linux-arm64.tar.gz.sha256 --file vz-linux-arm64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz --file vz-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-amd64.tar.gz.sha256 --file vz-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz --file vz-darwin-arm64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/vz-darwin-arm64.tar.gz.sha256 --file vz-darwin-arm64.tar.gz.sha256
