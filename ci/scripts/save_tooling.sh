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

cp ../scripts/k8s-dump-cluster.sh ${WORKSPACE}
cd out
tar -czf ${WORKSPACE}/verrazzano-analysis-linux-amd64.tar.gz -C linux_amd64 .
tar -czf ${WORKSPACE}/verrazzano-analysis-darwin-amd64.tar.gz -C darwin_amd64 .
cd ${WORKSPACE}
sha256sum k8s-dump-cluster.sh > k8s-dump-cluster.sh.sha256
sha256sum verrazzano-analysis-linux-amd64.tar.gz > verrazzano-analysis-linux-amd64.tar.gz.sha256
sha256sum verrazzano-analysis-darwin-amd64.tar.gz > verrazzano-analysis-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/k8s-dump-cluster.sh --file k8s-dump-cluster.sh
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/k8s-dump-cluster.sh.sha256 --file k8s-dump-cluster.sh.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/verrazzano-analysis-linux-amd64.tar.gz --file verrazzano-analysis-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/verrazzano-analysis-linux-amd64.tar.gz.sha256 --file verrazzano-analysis-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/verrazzano-analysis-darwin-amd64.tar.gz --file verrazzano-analysis-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CURRENT_BRANCH_NAME}/verrazzano-analysis-darwin-amd64.tar.gz.sha256 --file verrazzano-analysis-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/k8s-dump-cluster.sh --file k8s-dump-cluster.sh
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/k8s-dump-cluster.sh.sha256 --file k8s-dump-cluster.sh.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-linux-amd64.tar.gz --file verrazzano-analysis-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-linux-amd64.tar.gz.sha256 --file verrazzano-analysis-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-darwin-amd64.tar.gz --file verrazzano-analysis-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${CURRENT_BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-darwin-amd64.tar.gz.sha256 --file verrazzano-analysis-darwin-amd64.tar.gz.sha256
