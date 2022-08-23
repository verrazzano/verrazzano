#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
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
  echo "Bucket label for zip must be specified"
  exit 1
fi
BUCKET_LABEL="$3"

if [ -z "$4" ]; then
  echo "Root of Verrazzano repository must be specified"
  exit 1
fi
VZ_REPO_ROOT="$4"

if [ -z "$5" ]; then
  echo "Path to the generated BOM file must be specified"
  exit 1
fi
GENERATED_BOM_FILE="$5"

if [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ] || [ -z "$VERRAZZANO_DEV_VERSION" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi


# Create the general distribution layout under a given root directory
createDistributionLayout() {
  local distributionDirectory=$1
  echo "Creating the parent directory ${distributionDirectory} for the distribution layout ..."
  mkdir -p ${distributionDirectory}
  chmod uog+w ${distributionDirectory}

  mkdir -p ${distributionDirectory}/bin
  mkdir -p ${distributionDirectory}/manifests/k8s
  mkdir -p ${distributionDirectory}/manifests/charts
  mkdir -p ${distributionDirectory}/manifests/profiles
}

# Download the artifacts which are already built and common to both open-source distribution and commercial distribution
downloadCommonFiles() {
  mkdir -p ${VZ_DISTRIBUTION_COMMON}
  echo "Downloading common artifacts under ${VZ_DISTRIBUTION_COMMON} ..."

  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/operator.yaml --file ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml

  # Verrazzano CLI for Linux AMD64
  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_LINUX_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ}
  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_LINUX_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ_SHA256}

  # Verrazzano CLI for Darwin AMD64
  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_DARWIN_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ}
  oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256}

  # Do we need SHA-256 for CLI in the distribution ?
}

# Copy profiles from the source repository to the directory from where the distribution bundles will be built
copyProfiles() {
  local profileDirectory=$1
  echo "Copying profiles to ${profileDirectory} ..."

  # Copy samples profiles from the source repository
  cp ${VZ_REPO_ROOT}/platform-operator/config/samples/install-default.yaml ${profileDirectory}/default.yaml
  cp ${VZ_REPO_ROOT}/platform-operator/config/samples/install-dev.yaml ${profileDirectory}/dev.yaml
  cp ${VZ_REPO_ROOT}/platform-operator/config/samples/install-managed-cluster.yaml ${profileDirectory}/managed-cluster.yaml
  cp ${VZ_REPO_ROOT}/platform-operator/config/samples/install-oci.yaml ${profileDirectory}/oci.yaml
  cp ${VZ_REPO_ROOT}/platform-operator/config/samples/install-ocne.yaml ${profileDirectory}/ocne.yaml
}

# Copy profiles from the source repository to the directory from where the distribution bundles will be built
generateOpenSourceDistribution() {
  mkdir -p ${VZ_DISTRIBUTION_GENERATED}

  cp ${VZ_REPO_ROOT}/LICENSE.txt ${VZ_OPENSOURCE_ROOT}/LICENSE

  # Include README.md and README.html

  # vz-registry-image-helper.sh has a dependency on bom_utils.sh, so copy both the files
  cp ${VZ_REPO_ROOT}/tools/scripts/vz-registry-image-helper.sh ${VZ_OPENSOURCE_ROOT}/bin/vz-registry-image-helper.sh
  cp ${VZ_REPO_ROOT}/tools/scripts/bom_utils.sh ${VZ_OPENSOURCE_ROOT}/bin/bom_utils.sh
  # Defer downloading the CLI to the end, just before creating the distribution bundle

  # Copy operator.yaml and charts
  cp ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml ${VZ_OPENSOURCE_ROOT}/manifests/k8s/verrazzano-platform-operator.yaml
  cp -r ${VZ_REPO_ROOT}/platform-operator/helm_config/charts/verrazzano-platform-operator ${VZ_OPENSOURCE_ROOT}/manifests/charts

  # Copy profiles
  copyProfiles ${VZ_OPENSOURCE_ROOT}/manifests/profiles

  # Copy Bill Of Materials, containing the list of images
  cp ${GENERATED_BOM_FILE} ${VZ_OPENSOURCE_ROOT}/manifests/verrazzano-bom.json

  # Extract the CLI for Linux AMD64
  echo "Extract the CLI for Linux AMD64 ..."
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ} -C ${VZ_OPENSOURCE_ROOT}/bin

  # Build distribution for Linux AMD64 architecture
  echo "Build distribution for Linux AMD64 architecture ..."
  tar -czf ${VZ_DISTRIBUTION_GENERATED}/${VZ_LINUX_AMD64_TARGZ} -C ${VZ_OPENSOURCE_ROOT} .
  sha256sum ${VZ_DISTRIBUTION_GENERATED}/${VZ_LINUX_AMD64_TARGZ} > ${VZ_DISTRIBUTION_GENERATED}/${VZ_LINUX_AMD64_TARGZ_SHA256}

  # Clean-up CLI for Linux AMD64 and extract CLI for Darwin AMD64 architecture
  echo "Clean-up CLI for Linux AMD64 and extract CLI for Darwin AMD64 architecture ..."
  rm -f ${VZ_OPENSOURCE_ROOT}/bin/vz
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ} -C ${VZ_OPENSOURCE_ROOT}/bin

  # Build distribution for Darwin AMD64 architecture
  echo "Build distribution for Darwin AMD64 architecture ..."
  tar -czf ${VZ_DISTRIBUTION_GENERATED}/${VZ_DARWIN_AMD64_TARGZ} -C ${VZ_OPENSOURCE_ROOT} .
  sha256sum ${VZ_DISTRIBUTION_GENERATED}/${VZ_DARWIN_AMD64_TARGZ} > ${VZ_DISTRIBUTION_GENERATED}/${VZ_DARWIN_AMD64_TARGZ_SHA256}
}

# Upload the generated distribution bundles to object store
uploadOpenSourceDistribution() {
  oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_LINUX_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_GENERATED}/${VZ_LINUX_AMD64_TARGZ}
  oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_LINUX_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_GENERATED}/${VZ_LINUX_AMD64_TARGZ_SHA256}
  oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_DARWIN_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_GENERATED}/${VZ_DARWIN_AMD64_TARGZ}
  oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_DARWIN_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_GENERATED}/${VZ_DARWIN_AMD64_TARGZ_SHA256}
}

# Clean-up workspace after uploading the distribution bundles
cleanupWorkspace() {
  rm -rf ${VZ_DISTRIBUTION_COMMON}
  rm -rf ${VZ_OPENSOURCE_ROOT}
  rm -rf ${VZ_DISTRIBUTION_GENERATED}
}

# List of files in storage
VZ_CLI_LINUX_AMD64_TARGZ="vz-linux-amd64.tar.gz"
VZ_CLI_LINUX_AMD64_TARGZ_SHA256="vz-linux-amd64.tar.gz.sha256"

VZ_CLI_DARWIN_AMD64_TARGZ="vz-darwin-amd64.tar.gz"
VZ_CLI_DARWIN_AMD64_TARGZ_SHA256="vz-darwin-amd64.tar.gz.sha256"

DISTRIBUTION_PREFIX="verrazzano-${VERRAZZANO_DEV_VERSION}"
echo "DISTRIBUTION_PREFIX ${DISTRIBUTION_PREFIX}"

VZ_LINUX_AMD64_TARGZ="${DISTRIBUTION_PREFIX}-linux-amd64.tar.gz"
VZ_LINUX_AMD64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-linux-amd64.tar.gz.sha256"

VZ_DARWIN_AMD64_TARGZ="${DISTRIBUTION_PREFIX}-darwin-amd64.tar.gz"
VZ_DARWIN_AMD64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-darwin-amd64.tar.gz.sha256"

# Directory to contain the files which are common for both types of distribution bundles
VZ_DISTRIBUTION_COMMON="${WORKSPACE}/vz-distribution-common"

# Directory to hold the generated distribution bundles
VZ_DISTRIBUTION_GENERATED="${WORKSPACE}/vz-distribution-generated"

# Directory containing the layout and required files for the open-source distribution
VZ_OPENSOURCE_ROOT="${WORKSPACE}/vz-open-source"

# Call the function to download the artifacts common to both types of distribution bundles
downloadCommonFiles

# Build open-source distribution bundles
createDistributionLayout "${VZ_OPENSOURCE_ROOT}"
generateOpenSourceDistribution "${VZ_OPENSOURCE_ROOT}"

# Common for both types of distribution bundles
uploadOpenSourceDistribution
cleanupWorkspace