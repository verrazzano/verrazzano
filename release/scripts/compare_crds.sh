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
    $(basename $0) <from> <to>

  Example:
    $(basename $0) v1.0.0 HEAD

  This script should be run from the git repository containing the bits to release. "from" and "to" can be tags, commits, etc.
  "from" defaults to the most recent version tag and "to" defaults to HEAD.
EOM
    exit 0
}

[ "$1" == "-h" ] && { usage; }

FROM=$1
TO=${2:-HEAD}

# Default to the latest tag
if [[ -z "$FROM" ]]; then
   git fetch --tags
   FROM=$(git tag --sort=taggerdate | tail -1)
fi

echo "Showing all CRD differences between $FROM and $TO"
echo ""

SCRIPT_DIR=$(dirname "$0")
git --no-pager diff --exit-code $FROM $TO -- `find $SCRIPT_DIR/../.. -type f -path '*/crds/*.yaml'`
