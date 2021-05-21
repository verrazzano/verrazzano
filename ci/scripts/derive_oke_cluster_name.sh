#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$BRANCH_NAME" ] || [ -z "$SHORT_TIME_STAMP" ] || [ -z "$BUILD_NUMBER" ] ; then
  echo "This script must only be called from Jenkins and requires environment variables BRANCH_NAME, SHORT_TIME_STAMP and BUILD_NUMBER are set"
  exit 1
fi

# The prefix for the OKE cluster is derived using the BRANCH_NAME, SHORT_TIME_STAMP and BUILD_NUMBER as below
# <first 8 alpha numeric characters from the branch><5 digit build number>-<8 digit timestamp>
#
# The script truncates if any of these values are more than the defined numbers. The current time stamp is not
# derived in the script, to allow the script to derive the same CLUSTER_PREFIX on repeated calls.
#

NEW_BRANCH=$(echo "$BRANCH_NAME" | sed 's/[^a-zA-Z0-9]//g')
NEW_BRANCH=${NEW_BRANCH:0:8}

CLUSTER_PREFIX="$NEW_BRANCH$BUILD_NUMBER"

if (( ${#CLUSTER_PREFIX} > 13 )); then
  CLUSTER_PREFIX=${CLUSTER_PREFIX:0:13}
fi

TIMESTAMP=${SHORT_TIME_STAMP}
if (( ${#TIMESTAMP} > 8 )); then
  TIMESTAMP=${TIMESTAMP:0:8}
fi

CLUSTER_PREFIX="$CLUSTER_PREFIX-$TIMESTAMP"
echo "$CLUSTER_PREFIX"