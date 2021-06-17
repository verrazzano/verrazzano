#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$SHORT_COMMIT_HASH" ] ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cd out
tar -czf ${WORKSPACE}/verrazzano-analysis-linux-amd64.tar.gz -C linux_amd64 .
tar -czf ${WORKSPACE}/verrazzano-analysis-darwin-amd64.tar.gz -C darwin_amd64 .
cd ${WORKSPACE}
sha256sum verrazzano-analysis-linux-amd64.tar.gz > verrazzano-analysis-linux-amd64.tar.gz.sha256
sha256sum verrazzano-analysis-darwin-amd64.tar.gz > verrazzano-analysis-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/verrazzano-analysis-linux-amd64.tar.gz --file verrazzano-analysis-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/verrazzano-analysis-linux-amd64.tar.gz.sha256 --file verrazzano-analysis-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/verrazzano-analysis-darwin-amd64.tar.gz --file verrazzano-analysis-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/verrazzano-analysis-darwin-amd64.tar.gz.sha256 --file verrazzano-analysis-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/verrazzano-analysis-linux-amd64.tar.gz --file verrazzano-analysis-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/verrazzano-analysis-linux-amd64.tar.gz.sha256 --file verrazzano-analysis-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/verrazzano-analysis-darwin-amd64.tar.gz --file verrazzano-analysis-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/verrazzano-analysis-darwin-amd64.tar.gz.sha256 --file verrazzano-analysis-darwin-amd64.tar.gz.sha256
