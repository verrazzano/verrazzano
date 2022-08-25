#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "$1" ]; then
  echo "Root of Verrazzano repository must be specified"
  exit 1
fi
VZ_REPO_ROOT="$1"

if [ -z "$2" ]; then
  echo "Path to the generated BOM file must be specified"
  exit 1
fi
GENERATED_BOM_FILE="$2"

if [ -z "$3" ]; then
  echo "Verrazzano development version must be specified"
  exit 1
fi
VZ_DEVELOPENT_VERSION="$3"

if [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ]  || [ -z "$OCI_OS_REGION" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi


# Create the general distribution layout under a given root directory
createDistributionLayout() {
  local distributionDirectory=$1
  echo "Creating the distribution layout under ${distributionDirectory} ..."
  mkdir -p ${distributionDirectory}
  chmod uog+w ${distributionDirectory}

  mkdir -p ${distributionDirectory}/bin
  mkdir -p ${distributionDirectory}/manifests/k8s
  mkdir -p ${distributionDirectory}/manifests/charts
  mkdir -p ${distributionDirectory}/manifests/profiles

  if [ "${distributionDirectory}" == "${VZ_COMMERCIAL_ROOT}" ];then
     echo "Creating the directory to place images and CLIs for supported platforms for commercial distribution ..."
     # Create a directory to place the images
     mkdir -p ${distributionDirectory}/images

     # Directory to place the CLI
     mkdir -p ${distributionDirectory}/bin/darwin-amd64
     mkdir -p ${distributionDirectory}/bin/darwin-arm64
     mkdir -p ${distributionDirectory}/bin/linux-amd64
     mkdir -p ${distributionDirectory}/bin/linux-arm64
  fi
}

# Download the artifacts which are already built and common to both open-source distribution and commercial distribution
downloadCommonFiles() {
  echo "Downloading common artifacts under ${VZ_DISTRIBUTION_COMMON} ..."
  mkdir -p ${VZ_DISTRIBUTION_COMMON}

  # operator.yaml
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/operator.yaml --file ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml

  # CLI for Linux AMD64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_LINUX_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_LINUX_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ_SHA256}

  # CLI for Linux ARM64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_LINUX_ARM64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_ARM64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_LINUX_ARM64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_ARM64_TARGZ_SHA256}

  # CLI for Darwin AMD64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_DARWIN_AMD64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ_SHA256}

  # CLI for Darwin ARM64
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_DARWIN_ARM64_TARGZ} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_ARM64_TARGZ}
  oci --region ${OCI_OS_REGION} os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_CLI_DARWIN_ARM64_TARGZ_SHA256} --file ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_ARM64_TARGZ_SHA256}
}

# Copy the common files to directory from where the script builds Verrazzano release distribution
includeCommonFiles() {
  local distributionDirectory=$1
  cp ${VZ_REPO_ROOT}/LICENSE.txt ${distributionDirectory}/LICENSE

  # Include README.md and README.html

  # vz-registry-image-helper.sh has a dependency on bom_utils.sh, so copy both the files
  cp ${VZ_REPO_ROOT}/tools/scripts/vz-registry-image-helper.sh ${distributionDirectory}/bin/vz-registry-image-helper.sh
  cp ${VZ_REPO_ROOT}/tools/scripts/bom_utils.sh ${distributionDirectory}/bin/bom_utils.sh

  # Copy operator.yaml and charts
  cp ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml ${distributionDirectory}/manifests/k8s/verrazzano-platform-operator.yaml
  cp -r ${VZ_REPO_ROOT}/platform-operator/helm_config/charts/verrazzano-platform-operator ${distributionDirectory}/manifests/charts
  rm -f ${distributionDirectory}/manifests/charts/verrazzano-platform-operator/.helmignore || true

  # Copy profiles
  copyProfiles ${distributionDirectory}/manifests/profiles

  # Copy Bill Of Materials, containing the list of images
  cp ${GENERATED_BOM_FILE} ${distributionDirectory}/manifests/verrazzano-bom.json
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

# Create a text file containing the contents of the bundle
captureBundleContents() {
  local rootDir=$1
  local generatedDir=$2
  local textFile=$3

  cd ${rootDir}
  find * -type f > "${generatedDir}/${textFile}"
  if [ -f "${generatedDir}/${textFile}" ];
  then
    echo "Sorting file ${generatedDir}/${textFile}"
    sort -u -o "${generatedDir}/${textFile}" "${generatedDir}/${textFile}"
  fi
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${OS_LINUX_AMD64_BUNDLE_CONTENTS} --file ${generatedDir}/${textFile}
  rm ${generatedDir}/${textFile}
}

# Generate the open-source Verrazzano release distribution
generateOpenSourceDistribution() {
  echo "Generate open-source distribution ..."
  local rootDir=$1
  local generatedDir=$2

  mkdir -p ${generatedDir}
  includeCommonFiles $rootDir

  # Extract the CLI for Linux AMD64
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ} -C ${rootDir}/bin

  # Build distribution for Linux AMD64 architecture
  echo "Build distribution for Linux AMD64 architecture ..."
  tar -czf ${generatedDir}/${VZ_LINUX_AMD64_TARGZ} -C ${rootDir} .

  captureBundleContents ${rootDir} ${generatedDir} ${OS_LINUX_AMD64_BUNDLE_CONTENTS}

  # Clean-up CLI for Linux AMD64 and extract CLI for Darwin AMD64 architecture
  echo "Clean-up CLI for Linux AMD64 and extract CLI for Darwin AMD64 architecture ..."
  rm -f ${rootDir}/bin/vz
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ} -C ${rootDir}/bin

  # Build distribution for Darwin AMD64 architecture
  tar -czf ${generatedDir}/${VZ_DARWIN_AMD64_TARGZ} -C ${rootDir} .

  captureBundleContents ${rootDir} ${generatedDir} ${OS_DARWIN_AMD64_BUNDLE_CONTENTS}

  cp ${VZ_DISTRIBUTION_COMMON}/verrazzano-platform-operator.yaml ${generatedDir}/operator.yaml

  cd ${generatedDir}
  sha256sum ${VZ_LINUX_AMD64_TARGZ} > ${VZ_LINUX_AMD64_TARGZ_SHA256}
  sha256sum ${VZ_DARWIN_AMD64_TARGZ} > ${VZ_DARWIN_AMD64_TARGZ_SHA256}
  sha256sum operator.yaml > operator.yaml.sha256

  captureBundleContents ${generatedDir} ${generatedDir} ${OS_BUNDLE_CONTENTS}

  # Create and upload the final distribution zip file and upload
  echo "Build open-source distribution ${generatedDir}/${VZ_OPENSOURCE_RELEASE_BUNDLE} ..."

  zip ${VZ_OPENSOURCE_RELEASE_BUNDLE} ${VZ_LINUX_AMD64_TARGZ} ${VZ_LINUX_AMD64_TARGZ_SHA256} ${VZ_DARWIN_AMD64_TARGZ} ${VZ_DARWIN_AMD64_TARGZ_SHA256} operator.yaml operator.yaml.sha256
  sha256sum ${VZ_OPENSOURCE_RELEASE_BUNDLE} > ${VZ_OPENSOURCE_RELEASE_BUNDLE_SHA256}

  echo "Upload open-source distribution ${generatedDir}/${VZ_OPENSOURCE_RELEASE_BUNDLE} ..."
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_OPENSOURCE_RELEASE_BUNDLE} --file ${VZ_OPENSOURCE_RELEASE_BUNDLE}
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_OPENSOURCE_RELEASE_BUNDLE_SHA256} --file ${VZ_OPENSOURCE_RELEASE_BUNDLE_SHA256}
  echo "Successfully uploaded ${generatedDir}/${VZ_OPENSOURCE_RELEASE_BUNDLE}"
}

# Generate the commercial Verrazzano release distribution
generateCommercialDistribution() {
  echo "Generate commercial distribution ..."
  local rootDir=$1
  local generatedDir=$2

  mkdir -p ${generatedDir}
  includeCommonFiles "${rootDir}"

  # Extract the CLIs for supported architectures
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_AMD64_TARGZ} -C ${rootDir}/bin/linux-amd64
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_LINUX_ARM64_TARGZ} -C ${rootDir}/bin/linux-arm64

  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_AMD64_TARGZ} -C ${rootDir}/bin/darwin-amd64
  tar xzf ${VZ_DISTRIBUTION_COMMON}/${VZ_CLI_DARWIN_ARM64_TARGZ} -C ${rootDir}/bin/darwin-arm64

  # Move the tar files to images directory
  mv ${WORKSPACE}/tar-files/*.tar ${rootDir}/images/

  captureBundleContents ${rootDir} ${generatedDir} ${COMM_BUNDLE_CONTENTS}

  # Create and upload the final distribution zip file and upload
  echo "Create ${generatedDir}/${VZ_COMMERCIAL_RELEASE_BUNDLE} and upload ..."

  zip -r ${generatedDir}/${VZ_COMMERCIAL_RELEASE_BUNDLE} *
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_COMMERCIAL_RELEASE_BUNDLE} --file ${generatedDir}/${VZ_COMMERCIAL_RELEASE_BUNDLE}

  cd ${generatedDir}
  sha256sum ${VZ_COMMERCIAL_RELEASE_BUNDLE} > ${VZ_COMMERCIAL_RELEASE_BUNDLE_SHA256}
  oci --region ${OCI_OS_REGION} os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}-last-clean-periodic-test/${VZ_COMMERCIAL_RELEASE_BUNDLE_SHA256} --file ${VZ_COMMERCIAL_RELEASE_BUNDLE_SHA256}
  echo "Successfully uploaded ${generatedDir}/${VZ_COMMERCIAL_RELEASE_BUNDLE}"
}

# Clean-up workspace after uploading the distribution bundles
cleanupWorkspace() {
  rm -rf ${VZ_DISTRIBUTION_COMMON}
  rm -rf ${VZ_OPENSOURCE_ROOT}
  # Do not delete ${VZ_COMMERCIAL_ROOT} as push_to_ocir.sh requires ${VZ_COMMERCIAL_ROOT}/images/*.tar
  rm -rf ${VZ_OPENSOURCE_GENERATED}
  rm -rf ${VZ_COMMERCIAL_GENERATED}
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
VZ_OPENSOURCE_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}-open-source.zip"
VZ_OPENSOURCE_RELEASE_BUNDLE_SHA256="${VZ_OPENSOURCE_RELEASE_BUNDLE}.sha256"

VZ_COMMERCIAL_RELEASE_BUNDLE="${DISTRIBUTION_PREFIX}-commercial.zip"
VZ_COMMERCIAL_RELEASE_BUNDLE_SHA256="${VZ_COMMERCIAL_RELEASE_BUNDLE}.sha256"

# Linux AMD64 and Darwin AMD64 bundles for the open-source distribution
VZ_LINUX_AMD64_TARGZ="${DISTRIBUTION_PREFIX}-linux-amd64.tar.gz"
VZ_LINUX_AMD64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-linux-amd64.tar.gz.sha256"

VZ_DARWIN_AMD64_TARGZ="${DISTRIBUTION_PREFIX}-darwin-amd64.tar.gz"
VZ_DARWIN_AMD64_TARGZ_SHA256="${DISTRIBUTION_PREFIX}-darwin-amd64.tar.gz.sha256"

# Directory to contain the files which are common for both types of distribution bundles
VZ_DISTRIBUTION_COMMON="${WORKSPACE}/vz-distribution-common"

# Directory containing the layout and required files for the open-source distribution
VZ_OPENSOURCE_ROOT="${WORKSPACE}/vz-open-source"
VZ_OPENSOURCE_GENERATED="${WORKSPACE}/vz-open-source-generated"

# Directory containing the layout and required files for the commercial distribution
VZ_COMMERCIAL_ROOT="${WORKSPACE}/vz-commercial"
VZ_COMMERCIAL_GENERATED="${WORKSPACE}/vz-commercial-generated"

OS_LINUX_AMD64_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-open-source-linux-amd64.txt"
OS_DARWIN_AMD64_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-open-source-darwin-amd64.txt"
OS_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-open-source.txt"
COMM_BUNDLE_CONTENTS="${DISTRIBUTION_PREFIX}-commercial.txt"

# Call the function to download the artifacts common to both types of distribution bundles
downloadCommonFiles

# Build open-source distribution bundles
createDistributionLayout "${VZ_OPENSOURCE_ROOT}"
generateOpenSourceDistribution "${VZ_OPENSOURCE_ROOT}" "${VZ_OPENSOURCE_GENERATED}"

# Build commercial distribution bundle
createDistributionLayout "${VZ_COMMERCIAL_ROOT}"
generateCommercialDistribution "${VZ_COMMERCIAL_ROOT}" "${VZ_COMMERCIAL_GENERATED}"

# Delete the directories created under WORKSPACE
cleanupWorkspace