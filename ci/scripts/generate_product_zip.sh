#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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
  echo "The tar/Zip file prefix must be specified"
  exit 1
fi
ZIPFILE_PREFIX="$4"

if [ -z "$5" ]; then
  echo "Path to the generated BOM file must be specified"
  exit 1
fi
GENERATED_BOM_FILE="$5"

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

mkdir ${WORKSPACE}/tar-files
chmod uog+w ${WORKSPACE}/tar-files
cp $GENERATED_BOM_FILE ${WORKSPACE}/tar-files/verrazzano-bom.json
cp tools/scripts/bom_utils.sh ${WORKSPACE}/tar-files/bom_utils.sh
cp tools/scripts/vz-registry-image-helper.sh ${WORKSPACE}/tar-files/vz-registry-image-helper.sh
cp tools/scripts/README.md ${WORKSPACE}/tar-files/README.md
mkdir -p ${WORKSPACE}/tar-files/charts
cp  -r platform-operator/helm_config/charts/verrazzano-platform-operator ${WORKSPACE}/tar-files/charts

tarfile="${ZIPFILE_PREFIX}.tar.gz"
commitFile="${ZIPFILE_PREFIX}-commit.txt"
sha256File="${tarfile}.sha256"
zipFile="${ZIPFILE_PREFIX}.zip"
readmeFile="readme.txt"

cat <<EOF > ${WORKSPACE}/${readmeFile}
Verrazzano Enterprise Container Platform archive for private registry install.

See https://verrazzano.io/latest/docs/setup/private-registry/private-registry for details.
EOF

# tools/scripts/generate_tarball.sh ${WORKSPACE}/tar-files/verrazzano-bom.json ${WORKSPACE}/tar-files ${WORKSPACE}/${tarfile}
# cd ${WORKSPACE}
# sha256sum ${tarfile} > ${sha256File}
# echo "git-commit=${GIT_COMMIT_USED}" > ${commitFile}
# oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${BUCKET_LABEL}/${commitFile} --file ${commitFile}
# zip ${zipFile} ${commitFile} ${sha256File} ${readmeFile} ${tarfile}
# oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${BUCKET_LABEL}/${zipFile} --file ${zipFile}
