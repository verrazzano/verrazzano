#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Downloads the verrazzano-platform-operator.yaml, the Verrazzano distributions for AMD64 and ARM64 architectures.

set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Downloads the verrazzano-platform-operator.yaml, the Verrazzano distributions for AMD64 and ARM64 architectures.

  Usage:
    $(basename $0) <release branch> <short hash of commit to release> <release bundle> <directory where the release artifacts need to be downloaded> [VERIFY-ONLY]

  Example:
    $(basename $0) release-1.4 ab12123 verrazzano-1.4.0-lite.zip

  The script expects the OCI CLI is installed. It also expects the following environment variables -
    OCI_REGION - OCI region
    OBJECT_STORAGE_NS - top-level namespace used for the request
    OCI_OS_COMMIT_BUCKET - object storage bucket where the artifacts are stored
EOM
    exit 0
}

[ -z "$OCI_REGION" ] || [ -z "$OBJECT_STORAGE_NS" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ "$1" == "-h" ] && { usage; }

if [ -z "$1" ]; then
  echo "Verrazzano release branch is required"
  exit 1
fi
BRANCH=$1

if [ -z "$2" ]; then
  echo "The short commit used to build the release distribution is required"
  exit 1
fi
RELEASE_COMMIT_SHORT=$2

if [ -z "$3" ]; then
  echo "Verrazzano distribution to download is required"
  exit 1
fi
RELEASE_BUNDLE="$3"

if [ -z "$4" ]; then
  echo "Directory to download into is required"
  exit 1
fi
RELEASE_BINARIES_DIR=${4}

function verify_vz_release_artifacts_exist() {
    echo "Verifying release artifacts exist"
    if [ $# -ne 1 ] && [ $# -ne 2 ]; then
      echo "Usage: ${FUNCNAME[0]} commit release_bundle"
      return 1
    fi

    local _folder="$1"
    local _file="$2"
    oci --region ${OCI_REGION} os object head \
            --namespace ${OBJECT_STORAGE_NS} \
            --bucket-name ${OCI_OS_COMMIT_BUCKET} \
            --name "${_folder}/${_file}"
    if [ $? -ne 0 ]; then
        echo "${_folder}/${_file} was not found"
        exit 1
    fi

    oci --region ${OCI_REGION} os object head \
            --namespace ${OBJECT_STORAGE_NS} \
            --bucket-name ${OCI_OS_COMMIT_BUCKET} \
            --name "${_folder}/${_file}.sha256"
    if [ $? -ne 0 ]; then
        echo "${_folder}/${_file}.sha256 was not found"
        exit 1
    fi
}

function get_vz_release_artifacts() {
    if [ $# -ne 1 ] && [ $# -ne 2 ]; then
      echo "Usage: ${FUNCNAME[0]} commit release_bundle"
      return 1
    fi

    echo "Getting release artifacts"
    cd $RELEASE_BINARIES_DIR
    local _folder="$1"
    local _file="$2"
    oci --region ${OCI_REGION} os object get \
            --namespace ${OBJECT_STORAGE_NS} \
            --bucket-name ${OCI_OS_COMMIT_BUCKET} \
            --name "${_folder}/${_file}" \
            --file "${_file}"

    oci --region ${OCI_REGION} os object get \
            --namespace ${OBJECT_STORAGE_NS} \
            --bucket-name ${OCI_OS_COMMIT_BUCKET} \
            --name "${_folder}/${_file}.sha256" \
            --file "${_file}.sha256"

    SHA256_CMD="sha256sum -c"
    if [ "$(uname)" == "Darwin" ]; then
      SHA256_CMD="shasum -a 256 -c"
    fi
    ${SHA256_CMD} ${_file}.sha256
    unzip ${_file}
    rm -f ${_file}
    rm -f ${_file}.sha256
}

# Validate OCI CLI
validate_oci_cli || exit 1

mkdir -p $RELEASE_BINARIES_DIR

if [ "$5" == "VERIFY-ONLY" ]; then
    # verify the release artifacts exist, do not actually download them
    verify_vz_release_artifacts_exist ephemeral/$BRANCH/$RELEASE_COMMIT_SHORT ${RELEASE_BUNDLE} || exit 1
else
    # Download the release artifacts
    get_vz_release_artifacts ephemeral/$BRANCH/$RELEASE_COMMIT_SHORT ${RELEASE_BUNDLE} || exit 1
fi
