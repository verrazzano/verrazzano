#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Downloads the operator.yaml and the zip file containing the analysis tool.

set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Downloads the operator.yaml, the Verrazzano distributions for AMD64 and ARM64 architectures.

  Usage:
    $(basename $0) <release branch> <short hash of commit to release> <release bundle> <directory where the release artifacts need to be downloaded, defaults to the current directory>

  Example:
    $(basename $0) release-1.4 ab12123 verrazzano-1.4.0-lite.zip

  The script expects the OCI CLI is installed. It also expects the following environment variables -
    OCI_REGION - OCI region
    OBJECT_STORAGE_NS - top-level namespace used for the request
    OCI_OS_COMMIT_BUCKET - object storage bucket where the artifacts are stored
EOM
    exit 0
}

[ -z "$OCI_REGION" ] || [ -z "$OBJECT_STORAGE_NS" ] || [ -z "$OCI_OS_COMMIT_BUCKET" ] || [ -z "$1" ] || [ -z "$2" ] || [ "$1" == "-h" ] && { usage; }

BRANCH=$1
RELEASE_COMMIT_SHORT=$2
RELEASE_BUNDLE=$3
RELEASE_BINARIES_DIR=${4:-$SCRIPT_DIR}

function get_vz_release_artifacts() {
    if [ $# -ne 1 ] && [ $# -ne 2 ]; then
      echo "Usage: ${FUNCNAME[0]} commit release_bundle"
      return 1
    fi
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
    echo "Listing of $RELEASE_BINARIES_DIR"
    ls
}

# Validate OCI CLI
validate_oci_cli || exit 1

mkdir -p $RELEASE_BINARIES_DIR

# Download the release artifacts
get_vz_release_artifacts ephemeral/$BRANCH/$RELEASE_COMMIT_SHORT ${RELEASE_BUNDLE} || exit 1
