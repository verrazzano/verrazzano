#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Verifies GitHub release artifacts.
set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/common-release.sh

usage() {
    cat <<EOM
  Downloads the release artifacts from GitHub and checks the SHA256 hash.

  Usage:
    $(basename $0) <release version to verify>

  Example:
    $(basename $0) v1.0.1
EOM
    exit 0
}

[ -z "$1" ] || [ "$1" == "-h" ] && { usage; }

VERSION=${1}

TMPDIR=$(mktemp -d)
trap 'rm -r "${TMPDIR}"' exit

# Configure sha command based on platform
SHA_CMD="sha256sum -c"

if [ "$(uname)" == "Darwin" ]; then
    SHA_CMD="shasum -a 256 -c"
fi

function verify_released_artifacts() {
  local releaseVersionDir=${TMPDIR}/release
  mkdir -p $releaseVersionDir
  cd $releaseVersionDir

  # Iterate the array containing the release artifacts and download all of them
  echo "Downloading release artifacts for ${VERSION}"
  for i in "${releaseArtifacts[@]}"
  do
    local url="https://github.com/verrazzano/verrazzano/releases/download/$VERSION/$i"
    curl -Ss -L --show-error --fail -o $i ${url} || { echo "Unable to download ${url}"; exit; }
  done
  ${SHA_CMD} operator.yaml.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-darwin-amd64.tar.gz.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-darwin-arm64.tar.gz.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-linux-amd64.tar.gz.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-linux-arm64.tar.gz.sha256

  # Latest tag is automatic, do we really need to check ? If required, better compare the files from the two directories
  local latestVersionDir=${TMPDIR}}/latest
  mkdir -p $latestVersionDir
  cd $latestVersionDir

  # Iterate the array containing the release artifacts and download all of them
  echo "Downloading release artifacts for latest"
  for i in "${releaseArtifacts[@]}"
  do
    local url="https://github.com/verrazzano/verrazzano/releases/latest/download/$i"
    curl -Ss -L --show-error --fail -o $i ${url} || { echo "Unable to download ${url}"; exit; }
  done
  ${SHA_CMD} operator.yaml.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-darwin-amd64.tar.gz.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-darwin-arm64.tar.gz.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-linux-amd64.tar.gz.sha256
  ${SHA_CMD} verrazzano-${RELEASE_VERSION}-linux-arm64.tar.gz.sha256
}

verify_released_artifacts
