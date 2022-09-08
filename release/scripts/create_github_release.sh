#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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
    $(basename $0) <hash of commit to release> <directory containing the release binaries> <a boolean to indicate test run, defaults to true>

  Example:
    $(basename $0) v1.0.1 aa94949a4e8e9b50bc0674035898f2579f2519cb ~/go/src/github.com/verrazzano/verrazzano/release

  The script assumes Github CLI is installed and login is performed to authenticate the Github account.

EOM
    exit 0
}

[ -z "$1" ] || [ -z "$2" ] || [ "$1" == "-h" ] && { usage; }

if [ -z "${RELEASE_VERSION}" ] ; then
    echo "The script requires environment variable RELEASE_VERSION, in the format major.minor.patch (for example, 1.0.3)"
    exit 1
fi

RELEASE_COMMIT=${1}
RELEASE_BINARIES_DIR=${2}
TEST_RUN=${3:-true}

VERSION=${RELEASE_VERSION}

if [[ $VERSION != v* ]] ; then
  VERSION="v${1}"
fi

function verify_release_binaries_exist() {
  for i in "${releaseArtifacts[@]}"
  do
    if [ ! -f $i ];then
      echo "Release artifact $i not found!"
      return 1
    fi
  done
  return 0
}

cd ${RELEASE_BINARIES_DIR}

# Validate the expected release artifacts are available under current directory
verify_release_binaries_exist || exit 1

# Validate Github CLI
validate_github_cli || exit 1

if [ $TEST_RUN == true ] ; then
    echo "TEST_RUN is set to true, NOT doing a github release. This is the command that would be executed:"
    echo ""
    echo gh release create \"${VERSION}\" \
      --draft \
      --target \"${RELEASE_COMMIT}\" \
      --notes \"\" \
      --title \"Verrazzano release ${VERSION}\" \
    ${releaseArtifacts[*]}
else
    echo "TEST_RUN is set to false, doing a github release now."
    # Setting an empty string for notes, as the release notes will be prepared separately
    gh release create "${VERSION}" \
      --draft \
      --target "${RELEASE_COMMIT}" \
      --notes "" \
      --title "Verrazzano release ${VERSION}" \
    ${releaseArtifacts[*]}
fi
