#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# A script to archive the source as a .tar.gz file. When the size of the source archive is more than 4GB, it is split into
# parts such that each part is not more than 4GB

set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z "$1" ]; then
  echo "Specify the directory containing the source files"
  exit 1
fi

if [ -z "$2" ]; then
  echo "Specify the directory in which the source archive is to be created"
  exit 1
fi

if [ -z "$3" ]; then
  echo "Specify the Verrazzano release version"
  exit 1
fi

SOURCE_DIR=$1
ARCHIVE_DIR=$2
VERRAZZANO_VERSION=$3

VERRAZZANO_PREFIX="verrazzano"
VERRAZZANO_RELEASE=${VERRAZZANO_PREFIX}-${VERRAZZANO_VERSION}
SOURCE_ARCHIVE=${VERRAZZANO_RELEASE}.tar.gz

# Create an archive from source code in SOURCE_DIR
function createSourceArchive() {
  cd $SOURCE_DIR
  tar czf ${ARCHIVE_DIR}/${SOURCE_ARCHIVE} .
}

# Split the generated tar file, such that each of the source code file has maximum of 4GB
# Resulting files will be ${VERRAZZANO_RELEASE}.part00.tar.gz, ${VERRAZZANO_RELEASE}.part01.tar.gz, etc
function splitSourceArchive() {
  cd ${ARCHIVE_DIR}
  if [ $(stat -c %s "${SOURCE_ARCHIVE}") -gt ${MAX_FILE_SIZE} ]; then
    echo "The source archive ${SOURCE_ARCHIVE} is of size more than 4GB, splitting it into multiple files and restrict the maximum size to 4GB"
    split -b 4G ${SOURCE_ARCHIVE} -d --additional-suffix=.tar.gz "${VERRAZZANO_RELEASE}.part"
    split_count=0
    for i in ${VERRAZZANO_RELEASE}.part*.tar.gz;
    do
      # Create sha256 for all the parts
      ${SHA256_CMD} ${i} > ${i}.sha256

      # Upload the file and the SHA to Object Storage
      split_count=$(($split_count+1))
    done
    echo "The source archive is split into ${split_count} files, make sure that they can be combined to form a .tar.gz file, which can be extracted"
    echo "Sample command to combine the parts: cat ${VERRAZZANO_RELEASE}.part*.tar.gz >${VERRAZZANO_RELEASE}_joined.tar.gz"
  else
    echo "Created source archive ${ARCHIVE_DIR}/${SOURCE_ARCHIVE}"
    sha256sum ${SOURCE_ARCHIVE} > ${SOURCE_ARCHIVE}.sha256
    # Upload the file and the SHA to Object Storage
  fi
}

# Command to calculate the SHA256
SHA256_CMD="sha256sum"
if [ "$(uname)" == "Darwin" ]; then
    SHA256_CMD="shasum -a 256"
fi

# Restrict the maximum size of the file to 4GB
MAX_FILE_SIZE=4294967296

# Create ARCHIVE_DIR, if it is not there already
if [[ ! -d "${ARCHIVE_DIR}" ]]; then
  mkdir -p ${ARCHIVE_DIR}
fi

createSourceArchive
splitSourceArchive
