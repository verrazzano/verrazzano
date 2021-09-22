#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Run prerelease validation checks

usage() {
    cat <<EOM
  Performs pre-release validation checks.

  Usage:
    $(basename $0) <version to release>

  Example:
    $(basename $0) 1.0.2

  This script depends on git and verrazzano-helper and should be run from the git repository containing the bits to release.
EOM
    exit 0
}

[ -z "$1" -o "$1" == "-h" ] && { usage; }

VERSION=$1
SCRIPT_DIR=$(dirname "$0")

# Check for CRD changes

echo "Checking for CRD changes... you should visually inspect for potential backward incompatibilities"
$SCRIPT_DIR/compare_crds.sh
EXIT_CODE=$?
echo ""

# Check .verrazzano-development-version and Chart.yaml versions

echo "Checking versions..."
$SCRIPT_DIR/check_versions.sh $VERSION
((EXIT_CODE |= $?))
echo ""

# If this is a patch release, check for any tickets that don't have backported commits
versionPattern='^([0-9]+\.)(\*|[0-9]+)$'
if [[ $VERSION =~ $versionPattern ]]; then
    echo "Not a patch release, skipping backported commits check"
else
    if ! command -v verrazzano-helper &> /dev/null
    then
      echo "verrazzano-helper must be in path"
      exit 1
    fi

    echo "Checking for missing backport commits..."
    verrazzano-helper get ticket-backports $VERSION --jira-env prod --token unused
    ((EXIT_CODE |= $?))
fi

# If the IGNORE_FAILURES environment variable is set, always exit with zero

[[ -n "$IGNORE_FAILURES" ]] && exit 0 || exit $EXIT_CODE
