#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Downloads the operator.yaml and the zip file containing the analysis tool.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Downloads the operator.yaml and the zip file containing the analysis tool.

  Usage:
    $(basename $0) <release branch> <short hash of commit to release> <directory where the release artifacts need to be downloaded, defaults to the current directory>

  Example:
    $(basename $0) release-1.0 ab12123

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
RELEASE_BINARIES_DIR=${3:-$SCRIPT_DIR}

function get_file_from_build_bucket() {
    if [ $# -ne 1 ] && [ $# -ne 2 ]; then
      echo "Usage: ${FUNCNAME[0]} commit [file]"
      return 1
    fi
    local _folder="$1"
    local _file="${2:--}"
    cd $RELEASE_BINARIES_DIR
    oci --region ${OCI_REGION} os object get \
            --namespace ${OBJECT_STORAGE_NS} \
            --bucket-name ${OCI_OS_COMMIT_BUCKET} \
            --name "${_folder}/${_file}" \
            --file "${_file}"
}

function get_vz_release_artifacts() {
    for i in "${releaseArtifacts[@]}"
    do
      get_file_from_build_bucket $1 $i
    done
}

# Validate OCI CLI
validate_oci_cli || exit 1

mkdir -p $RELEASE_BINARIES_DIR

# Download the release artifacts (note that master and release branch ephemeral retention is much longer than user branches)
get_vz_release_artifacts ephemeral/$BRANCH/$RELEASE_COMMIT_SHORT || exit 1
