#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Creates a Github release.
set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Creates a Github release.

  Usage:
    $(basename $0) <new version for the release> <hash of commit to release> <full path to the local Verrazzano git repository>

  Example:
    $(basename $0) v.1.0.1 aa94949a4e8e9b50bc0674035898f2579f2519cb ~/go/src/github.com/verrazzano/verrazzano

  The script assumes Github CLI is installed and login is performed to authenticate the Github account.

EOM
    exit 0
}

[ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ "$1" == "-h" ] && { usage; }

VERSION=${1}
RELEASE_COMMIT=${2}
GIT_REPO_DIRECTORY=${3}

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

function verify_release_artifacts() {
  local releaseVersionDir=$TMP_DIR/release
  mkdir -p $releaseVersionDir
  cd $releaseVersionDir

  # Iterate the array containing the release artifacts and download all of them
  for i in "${releaseArtifacts[@]}"
  do
    wget https://github.com/verrazzano/verrazzano/releases/download/$VERSION/$i
  done
  sha256sum -c k8s-dump-cluster.sh.sha256
  sha256sum -c verrazzano-analysis-darwin-amd64.tar.gz.sha256
  sha256sum -c verrazzano-analysis-linux-amd64.tar.gz.sha256

  # Latest tag is automatic, do we really need to check ? If required, better compare the files from the two directories
  local latestVersionDir=$TMP_DIR/latest
  mkdir -p $latestVersionDir
  cd $latestVersionDir

  # Iterate the array containing the release artifacts and download all of them
  for i in "${releaseArtifacts[@]}"
  do
    wget https://github.com/verrazzano/verrazzano/releases/latest/download/$i
  done
  sha256sum -c k8s-dump-cluster.sh.sha256
  sha256sum -c verrazzano-analysis-darwin-amd64.tar.gz.sha256
  sha256sum -c verrazzano-analysis-linux-amd64.tar.gz.sha256

  rm -rf $TMP_DIR
}

cd ${GIT_REPO_DIRECTORY}
validate_github_cli || exit 1

# Setting an empty string for notes, as the release notes will be prepared separately
gh release create "${VERSION}" \
 --target "${RELEASE_COMMIT}" \
 --notes "" \
 --title "Verrazzano release ${VERSION}" \
${OPERATOR_YAML} ${analysisToolFiles[*]}

verify_release_artifacts || exit 1
