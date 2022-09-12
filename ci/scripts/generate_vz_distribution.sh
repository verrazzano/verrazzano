#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

if [ -z "$1" ]; then
  echo "Root of Verrazzano repository must be specified"
  exit 1
fi
VZ_REPO_ROOT="$1"

if [ -z "$2" ]; then
  echo "Verrazzano development version must be specified"
  exit 1
fi
VZ_DEVELOPENT_VERSION="$2"

if [ -z "$3" ]; then
  echo "Short commit hash must be specified"
  exit 1
fi
SHORT_COMMIT_HASH_ENV="$3"

if [ -z "$BRANCH_NAME" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_REGION" ] || [ -z "$WORKSPACE" ]; then
  echo "This script requires environment variables - BRANCH_NAME, OCI_OS_COMMIT_BUCKET, OCI_OS_NAMESPACE, OCI_OS_REGION and WORKSPACE"
  exit 1
fi


# Create the general distribution layout under a given root directory
createDistributionLayout() {
  local rootDir=$1
  local devVersion=$2
  local distDir=${rootDir}/${devVersion}

  echo "Creating the distribution layout under ${distDir} ..."
  mkdir -p ${distDir}
  chmod uog+w ${distDir}

  mkdir -p ${distDir}/bin
  mkdir -p ${distDir}/manifests/k8s
  mkdir -p ${distDir}/manifests/charts
  mkdir -p ${distDir}/manifests/profiles

  if [ "${rootDir}" == "${VZ_FULL_ROOT}" ];then
     echo "Creating the directory to place images and CLIs for supported platforms for full distribution ..."
     # Create a directory to place the images
     mkdir -p ${distDir}/images

     # Directory to place the CLI
     mkdir -p ${distDir}/bin/darwin-amd64
     mkdir -p ${distDir}/bin/darwin-arm64
     mkdir -p ${distDir}/bin/linux-amd64
     mkdir -p ${distDir}/bin/linux-arm64
  fi
}

# Download the artifacts which are already built and common to both the distributions
downloadCommonFiles() {
  echo "Downloading common artifacts under ${VZ_DISTRIBUTION_COMMON} ..."
  mkdir -p ${VZ_DISTRIBUTION_COMMON}

  # operator.yaml
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/operator.yaml --file ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml

  # CLI for Linux AMD64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_LINUX_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_LINUX_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ_SHA256}

  # CLI for Linux ARM64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_LINUX_ARM64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_ARM64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_LINUX_ARM64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_ARM64_TARGZ_SHA256}

  # CLI for Darwin AMD64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_DARWIN_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256}

  # CLI for Darwin ARM64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_DARWIN_ARM64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_ARM64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_CLI_DARWIN_ARM64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_ARM64_TARGZ_SHA256}

  # Bill of materials
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/generated-verrazzano-bom.json --file ${VZ_DISTRIBUTION_COMMON}/verrazzano-bom.json

  # Validate SHA256 of the downloaded bundle
  SHA256_CMD="sha256sum -c"

  if [ "$(uname)" == "Darwin" ]; then
      SHA256_CMD="shasum -a 256 -c"
  fi
  cd ${VZ_DISTRIBUTION_COMMON}
  ${SHA256_CMD} ${VZ_CLI_LINUX_AMD64_TARGZ_SHA256}
  ${SHA256_CMD} ${VZ_CLI_LINUX_ARM64_TARGZ_SHA256}
  ${SHA256_CMD} ${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256}
  ${SHA256_CMD} ${VZ_CLI_DARWIN_ARM64_TARGZ_SHA256}
}

# Copy the common files to directory from where the script builds Verrazzano release distribution
includeCommonFiles() {
  local distDir=$1
  cp ${VZ_REPO_ROOT}/LICENSE.txt ${distDir}/LICENSE

  # vz-registry-image-helper.sh has a dependency on bom_utils.sh, so copy both the files
  cp ${VZ_REPO_ROOT}/tools/scripts/vz-registry-image-helper.sh ${distDir}/bin/vz-registry-image-helper.sh
  cp ${VZ_REPO_ROOT}/tools/scripts/bom_utils.sh ${distDir}/bin/bom_utils.sh

  # Copy operator.yaml and charts
  cp ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml ${distDir}/manifests/k8s/verrazzano-platform-operator.yaml
  cp -r ${VZ_REPO_ROOT}/platform-operator/helm_config/charts/verrazzano-platform-operator ${distDir}/manifests/charts

  # Copy profiles
  copyProfiles ${distributionDirectory}/manifests/profiles

  # Copy Bill Of Materials, containing the list of images
  cp ${VZ_DISTRIBUTION_COMMON}/verrazzano-bom.json ${distDir}/manifests/verrazzano-bom.json
}

# Copy profiles from the source repository to the directory from where the distribution bundles will be built
copyProfiles() {
  local profileDirectory=$1
  echo "Copying profiles to ${profileDirectory} ..."

  # Placeholder to copy profiles
}

# Create a text file containing the contents of the bundle
captureBundleContents() {
  local rootDir=$1
  local generatedDir=$2
  local textFile=$3

  cd ${rootDir}
  find * -type f > "${generatedDir}/${textFile}"
  if [ -f "${generatedDir}/${textFile}" ];
  then
    sort -u -o "${generatedDir}/${textFile}" "${generatedDir}/${textFile}"
  fi
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${textFile} --file ${generatedDir}/${textFile}
  rm ${generatedDir}/${textFile}
}

buildArchLiteBundle() {
  local vzCLI=$1
  local rootDir=$2
  local distDir=$3
  local generatedDir=$4
  local devVersion=$5
  local archLiteBundle=$6
  local textFile=$7

  # Extract the CLI for the given architecture
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${vzCLI} -C ${distDir}/bin

  # Copy readme
  cp ${VZ_REPO_ROOT}/release/docs/README_LITE.md ${distDir}/README.md

  # Build distribution for the given architecture
  cd ${rootDir}
  tar -czf ${generatedDir}/${archLiteBundle} ${devVersion}

  # Capture the contents of the bundle in a text file
  captureBundleContents ${rootDir} ${generatedDir} ${textFile}

  # Clean-up CLI
  rm -f ${distDir}/bin/vz
}

# Generate Verrazzano lite distribution
generateVZLiteDistribution() {
  echo "Generate Verrazzano lite distribution ..."
  local rootDir=$1
  local devVersion=$2
  local generatedDir=$3

  local distDir=${rootDir}/${devVersion}
  mkdir -p ${generatedDir}
  includeCommonFiles $distDir

  echo "Build distribution for Linux AMD64 architecture ..."
  buildArchLiteBundle ${VZ_CLI_LINUX_AMD64_TARGZ} ${rootDir} ${distDir} ${generatedDir} ${devVersion} ${VZ_LINUX_AMD64_TARGZ} ${LITE_LINUX_AMD64_BUNDLE_CONTENTS}

  echo "Build distribution for Linux ARM64 architecture ..."
  buildArchLiteBundle ${VZ_CLI_LINUX_ARM64_TARGZ} ${rootDir} ${distDir} ${generatedDir} ${devVersion} ${VZ_LINUX_ARM64_TARGZ} ${LITE_LINUX_ARM64_BUNDLE_CONTENTS}

  echo "Build distribution for Darwin AMD64 architecture ..."
  buildArchLiteBundle ${VZ_CLI_DARWIN_AMD64_TARGZ} ${rootDir} ${distDir} ${generatedDir} ${devVersion} ${VZ_DARWIN_AMD64_TARGZ} ${LITE_DARWIN_AMD64_BUNDLE_CONTENTS}

  echo "Build distribution for Darwin ARM64 architecture ..."
  buildArchLiteBundle ${VZ_CLI_DARWIN_ARM64_TARGZ} ${rootDir} ${distDir} ${generatedDir} ${devVersion} ${VZ_DARWIN_ARM64_TARGZ} ${LITE_DARWIN_ARM64_BUNDLE_CONTENTS}

  cp ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml ${generatedDir}/operator.yaml

  cd ${generatedDir}
  sha256sum ${VZ_LINUX_AMD64_TARGZ} > ${VZ_LINUX_AMD64_TARGZ_SHA256}
  sha256sum ${VZ_LINUX_ARM64_TARGZ} > ${VZ_LINUX_ARM64_TARGZ_SHA256}
  sha256sum ${VZ_DARWIN_AMD64_TARGZ} > ${VZ_DARWIN_AMD64_TARGZ_SHA256}
  sha256sum ${VZ_DARWIN_ARM64_TARGZ} > ${VZ_DARWIN_ARM64_TARGZ_SHA256}
  sha256sum operator.yaml > operator.yaml.sha256

  captureBundleContents ${generatedDir} ${generatedDir} ${LITE_BUNDLE_CONTENTS}

  # Create and upload the final distribution zip file and upload
  echo "Build Verrazzano lite distribution ${generatedDir}/${VZ_LITE_RELEASE_BUNDLE} ..."
  cd ${generatedDir}
  zip ${VZ_LITE_RELEASE_BUNDLE} *
  sha256sum ${VZ_LITE_RELEASE_BUNDLE} > ${VZ_LITE_RELEASE_BUNDLE_SHA256}

  echo "Upload Verrazzano lite distribution ${generatedDir}/${VZ_LITE_RELEASE_BUNDLE} ..."
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_LITE_RELEASE_BUNDLE} --file ${VZ_LITE_RELEASE_BUNDLE}
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_LITE_RELEASE_BUNDLE_SHA256} --file ${VZ_LITE_RELEASE_BUNDLE_SHA256}

  echo "Successfully uploaded ${generatedDir}/${VZ_LITE_RELEASE_BUNDLE}"
}

# Generate Verrazzano full release distribution
generateVZFullDistribution() {
  echo "Generate full distribution ..."
  local rootDir=$1
  local devVersion=$2
  local generatedDir=$3

  local distDir=${rootDir}/${devVersion}
  mkdir -p ${generatedDir}
  includeCommonFiles "${distDir}"

  # Extract the CLIs for supported architectures
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ} -C ${distDir}/bin/linux-amd64
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_ARM64_TARGZ} -C ${distDir}/bin/linux-arm64

  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ} -C ${distDir}/bin/darwin-amd64
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_ARM64_TARGZ} -C ${distDir}/bin/darwin-arm64

  # Create and upload the final distribution zip file and upload
  echo "Create ${generatedDir}/${VZ_FULL_RELEASE_BUNDLE} and upload ..."
  cp ${VZ_REPO_ROOT}/release/docs/README_FULL.md ${distDir}/README.md
  captureBundleContents ${rootDir} ${generatedDir} ${FULL_BUNDLE_CONTENTS}
  cd ${rootDir}
  zip -r ${generatedDir}/${VZ_FULL_RELEASE_BUNDLE} *
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_FULL_RELEASE_BUNDLE} --file ${generatedDir}/${VZ_FULL_RELEASE_BUNDLE}

  cd ${generatedDir}
  sha256sum ${VZ_FULL_RELEASE_BUNDLE} > ${VZ_FULL_RELEASE_BUNDLE_SHA256}
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${BRANCH_NAME}/${SHORT_COMMIT_HASH_ENV}/${VZ_FULL_RELEASE_BUNDLE_SHA256} --file ${VZ_FULL_RELEASE_BUNDLE_SHA256}

  echo "Successfully uploaded ${generatedDir}/${VZ_FULL_RELEASE_BUNDLE}"
}

# Download the tar files for the images defined in verrazzano-bom.json, and include them in full bundle
includeImageTarFiles() {
  local rootDir=$1
  local devVersion=$2
  local distDir=${rootDir}/${devVersion}/images
  ${VZ_REPO_ROOT}/tools/scripts/vz-registry-image-helper.sh -f ${distDir} -b ${VZ_DISTRIBUTION_COMMON}/verrazzano-bom.json
}

# generate profiles and remove cruft
includeProfiles() {
  local rootDir=$1
  local devVersion=$2
  local distDir=${rootDir}/${devVersion}/manifests/profiles
  export VERRAZZANO_ROOT=${VZ_REPO_ROOT}
  go run ${VZ_REPO_ROOT}/tools/generate-profiles/generate.go --profile prod --output-dir ${distDir}
  go run ${VZ_REPO_ROOT}/tools/generate-profiles/generate.go --profile dev --output-dir ${distDir}
  go run ${VZ_REPO_ROOT}/tools/generate-profiles/generate.go --profile managed-cluster --output-dir ${distDir}
  sanitizeProfiles ${distDir}/prod.yaml
  sanitizeProfiles ${distDir}/dev.yaml
  sanitizeProfiles ${distDir}/managed-cluster.yaml
}

sanitizeProfiles() {
  filePath=$1
  yq eval -i 'del(.status, .metadata.creationTimestamp)' ${filePath}
  cat ${filePath}
}
# Clean-up workspace after uploading the distribution bundles
cleanupWorkspace() {
  rm -rf ${VZ_DISTRIBUTION_COMMON}
  rm -rf ${VZ_LITE_ROOT}
  # Do not delete the generated files, which is required to push bundles to last_clean_periodic object
  # rm -rf ${VZ_LITE_GENERATED}
  # rm -rf ${VZ_FULL_GENERATED}
}

# List of files in storage
VZ_CLI_LINUX_AMD64_TARGZ="vz-linux-amd64.tar.gz"
VZ_CLI_LINUX_AMD64_TARGZ_SHA256="vz-linux-amd64.tar.gz.sha256"

VZ_CLI_LINUX_ARM64_TARGZ="vz-linux-arm64.tar.gz"
VZ_CLI_LINUX_ARM64_TARGZ_SHA256="vz-linux-arm64.tar.gz.sha256"

VZ_CLI_DARWIN_AMD64_TARGZ="vz-darwin-amd64.tar.gz"
VZ_CLI_DARWIN_AMD64_TARGZ_SHA256="vz-darwin-amd64.tar.gz.sha256"

VZ_CLI_DARWIN_ARM64_TARGZ="vz-darwin-arm64.tar.gz"
VZ_CLI_DARWIN_ARM64_TARGZ_SHA256="vz-darwin-arm64.tar.gz.sha256"

DISTRIBUTION_PREFIX="verrazzano-${VZ_DEVELOPENT_VERSION}"

# Release bundles and SHA256 of the bundles
VZ_LITE_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}-lite.zip"
VZ_LITE_RELEASE_BUNDLE_SHA256="${VZ_LITE_RELEASE_BUNDLE}.sha256"

VZ_FULL_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}.zip"
VZ_FULL_RELEASE_BUNDLE_SHA256="${VZ_FULL_RELEASE_BUNDLE}.sha256"

# Linux AMD64 and Darwin AMD64 bundles for the lite distribution
VZ_LINUX_AMD64_TARGZ="${DISTRIBUTION_PREFIX}-linux-amd64.tar.gz"
VZ_LINUX_AMD64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-linux-amd64.tar.gz.sha256"

VZ_DARWIN_AMD64_TARGZ="${DISTRIBUTION_PREFIX}-darwin-amd64.tar.gz"
VZ_DARWIN_AMD64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-darwin-amd64.tar.gz.sha256"

# Linux ARM64 and Darwin ARM64 bundles for the lite distribution
VZ_LINUX_ARM64_TARGZ="${DISTRIBUTION_PREFIX}-linux-arm64.tar.gz"
VZ_LINUX_ARM64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-linux-arm64.tar.gz.sha256"

VZ_DARWIN_ARM64_TARGZ="${DISTRIBUTION_PREFIX}-darwin-arm64.tar.gz"
VZ_DARWIN_ARM64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-darwin-arm64.tar.gz.sha256"

# Directory to contain the files which are common for both types of distribution bundles
VZ_DISTRIBUTION_COMMON="${WORKSPACE}/vz-distribution-common"

# Directory containing the layout and required files for the Verrazzano lite distribution
VZ_LITE_ROOT="${WORKSPACE}/vz-lite"
VZ_LITE_GENERATED="${WORKSPACE}/vz-lite-generated"

# Directory containing the layout and required files for the Verrazzano full distribution
VZ_FULL_ROOT="${WORKSPACE}/vz-full"
VZ_FULL_GENERATED="${WORKSPACE}/vz-full-generated"

LITE_LINUX_AMD64_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-lite-linux-amd64.txt"
LITE_LINUX_ARM64_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-lite-linux-arm64.txt"
LITE_DARWIN_AMD64_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-lite-darwin-amd64.txt"
LITE_DARWIN_ARM64_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-lite-darwin-arm64.txt"

LITE_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-lite.txt"
FULL_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-full.txt"

# Call the function to download the artifacts common to both types of distribution bundles
downloadCommonFiles
cd ${WORKSPACE}

# Build Verrazzano lite distribution bundles
createDistributionLayout "${VZ_LITE_ROOT}" "${DISTRIBUTION_PREFIX}"
generateVZLiteDistribution "${VZ_LITE_ROOT}" "${DISTRIBUTION_PREFIX}" "${VZ_LITE_GENERATED}"

# Build Verrazzano full distribution bundle
createDistributionLayout "${VZ_FULL_ROOT}" "${DISTRIBUTION_PREFIX}"
includeImageTarFiles "${VZ_FULL_ROOT}" "${DISTRIBUTION_PREFIX}"
includeProfiles "${VZ_FULL_ROOT}" "${DISTRIBUTION_PREFIX}"
generateVZFullDistribution "${VZ_FULL_ROOT}" "${DISTRIBUTION_PREFIX}" "${VZ_FULL_GENERATED}"

# Delete the directories created under WORKSPACE
cleanupWorkspace