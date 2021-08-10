#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

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

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/operator.yaml --file operator.yaml
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/operator.yaml --file operator.yaml
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/k8s-dump-cluster.sh --file k8s-dump-cluster.sh
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/k8s-dump-cluster.sh.sha256 --file k8s-dump-cluster.sh.sha256
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-linux-amd64.tar.gz --file verrazzano-analysis-linux-amd64.tar.gz
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-linux-amd64.tar.gz.sha256 --file verrazzano-analysis-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-darwin-amd64.tar.gz --file verrazzano-analysis-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/verrazzano-analysis-darwin-amd64.tar.gz.sha256 --file verrazzano-analysis-darwin-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/k8s-dump-cluster.sh --file k8s-dump-cluster.sh
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/k8s-dump-cluster.sh.sha256 --file k8s-dump-cluster.sh.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/verrazzano-analysis-linux-amd64.tar.gz --file verrazzano-analysis-linux-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/verrazzano-analysis-linux-amd64.tar.gz.sha256 --file verrazzano-analysis-linux-amd64.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/verrazzano-analysis-darwin-amd64.tar.gz --file verrazzano-analysis-darwin-amd64.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/verrazzano-analysis-darwin-amd64.tar.gz.sha256 --file verrazzano-analysis-darwin-amd64.tar.gz.sha256

# Generate a Verrazzano full Zip for private registry testing

# Get the latest stable generated BOM file
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/generated-verrazzano-bom.json --file ${WORKSPACE}/tar-files/verrazzano-bom.json
# Call the script to generate and publish the BOM
ci/scripts/generate_product_zip.sh ${env.GIT_COMMIT} ${SHORT_COMMIT_HASH} master-last-clean-periodic-test ${ZIPFILE_PREFIX} ${WORKSPACE}/tar-files/verrazzano-bom.json
