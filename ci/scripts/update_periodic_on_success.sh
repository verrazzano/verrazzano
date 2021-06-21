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

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

mkdir ${WORKSPACE}/tar-files
chmod uog+w ${WORKSPACE}/tar-files
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/${SHORT_COMMIT_HASH_ENV}/generated-verrazzano-bom.json --file ${WORKSPACE}/tar-files/verrazzano-bom.json
cp tools/scripts/vz-registry-image-helper.sh ${WORKSPACE}/tar-files/vz-registry-image-helper.sh
cp tools/scripts/README.md ${WORKSPACE}/tar-files/README.md
mkdir -p ${WORKSPACE}/tar-files/charts
cp  -r platform-operator/helm_config/charts/verrazzano-platform-operator ${WORKSPACE}/tar-files/charts
tools/scripts/generate_tarball.sh ${WORKSPACE}/tar-files/verrazzano-bom.json ${WORKSPACE}/tar-files ${WORKSPACE}/tarball.tar.gz
cd ${WORKSPACE}
sha256sum tarball.tar.gz > tarball.tar.gz.sha256
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/tarball.tar.gz --file tarball.tar.gz
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/tarball.tar.gz.sha256 --file tarball.tar.gz.sha256
echo "git-commit=${GIT_COMMIT_USED}" > tarball-commit.txt
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/tarball-commit.txt --file tarball-commit.txt
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
