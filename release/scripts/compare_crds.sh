#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Compare all CRD differences between two commits, tags, etc.

usage() {
    cat <<EOM
  Shows all CRD differences between two git tags, commits, etc.

  Usage:
    $(basename $0) <release version> <from> <to>

  Example:
    $(basename $0) 1.0.3

  This script should be run from the git repository containing the bits to release. "release version" is required.
  "from" and "to" can be tags, commits, etc. "from" defaults to the most recent version tag for the previous release and "to" defaults to HEAD.
EOM
    exit 0
}

[ "$1" == "-h" ] && { usage; }

VERSION=$1
FROM=$2
TO=${3:-HEAD}

# Default to the latest tag from the prior release
if [[ -z "$FROM" ]]; then
  MAJOR=$(echo ${VERSION} | cut -d. -f 1)
  MINOR=$(echo ${VERSION} | cut -d. -f 2)
  PATCH=$(echo ${VERSION} | cut -d. -f 3)

  git fetch --tags

  if [ "${MINOR}" == "0" ] && [ "${PATCH}" == "0" ]; then
    # Major version release - find the latest tag matching the previous major release
    PREV=v$(expr ${MAJOR} - "1").
  else
    if [ "${PATCH}" == "0" ]; then
      # Minor version release - find the latest tag matching the previous minor release
      PREV=v${MAJOR}.$(expr ${MINOR} - "1").
    else
      # Patch version release - find the latest tag matching the current major and minor version
      PREV=v${MAJOR}.${MINOR}.
    fi
  fi

  FROM=$(git tag --sort=taggerdate | grep ${PREV} | tail -1)
fi

echo "Showing all CRD differences between $FROM and $TO"
echo ""

SCRIPT_DIR=$(dirname "$0")
git --no-pager diff --exit-code $FROM $TO -- `find $SCRIPT_DIR/../.. -type f -path '*/crds/*.yaml'`
