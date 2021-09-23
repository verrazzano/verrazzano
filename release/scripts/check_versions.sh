#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Checks the Verrazzano development and Chart.yaml versions to make sure they are accurate

usage() {
    cat <<EOM
  Checks .verrazzano-development-version and Chart.yaml files to make sure the versions are correct.

  Usage:
    $(basename $0) <version to release>

  Example:
    $(basename $0) 1.0.2

  This script should be run from the git repository containing the bits to release.
EOM
    exit 0
}

[ -z "$1" -o "$1" == "-h" ] && { usage; }

VERSION=$1
PASS=true
SCRIPT_DIR=$(dirname "$0")

# Check .verrazzano-development-version

VER=$(grep verrazzano-development-version $SCRIPT_DIR/../../.verrazzano-development-version | cut -d= -f 2)

[[ "$VERSION" != "$VER" ]] && { echo "FAILED: .verrazzano-development-version has incorrect version"; PASS=false; }

# Check Chart.yaml versions

for f in $(find $SCRIPT_DIR/../../platform-operator/helm_config/charts/ -name Chart.yaml)
do
    VER=$(grep 'version:' $f | cut -d: -f 2 | xargs echo -n)
    [[ "$VERSION" != "$VER" ]] && { echo "FAILED: $f has incorrect version"; PASS=false; }

    VER=$(grep 'appVersion:' $f | cut -d: -f 2 | xargs echo -n)
    [[ "$VERSION" != "$VER" ]] && { echo "FAILED: $f has incorrect appVersion"; PASS=false; }
done

[[ "$PASS" == "true" ]] && { echo "All version checks passed"; exit 0; } || exit 1;
