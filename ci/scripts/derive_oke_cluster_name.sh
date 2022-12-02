#!/bin/bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -o pipefail

if [ -z "$BRANCH_NAME" ] || [ -z "$SHORT_TIME_STAMP" ] || [ -z "$BUILD_NUMBER" ] ; then
  echo "This script must only be called from Jenkins and requires environment variables BRANCH_NAME, SHORT_TIME_STAMP and BUILD_NUMBER are set."
  exit 1
fi

# The prefix for the OKE cluster is derived using the BRANCH_NAME, SHORT_TIME_STAMP and BUILD_NUMBER as below
# <8 or less alpha numeric characters from the branch><digit build number>-<10 digit timestamp>. The script truncates the
# branch name and the timestamp, if they contain more than the expected characters.

# WORKAROUNDS: A OKE TF module update changed the behaviour for the dns_labels. It started to use the label-prefix
# for those dns labels, rather than the short names it was using previously "cp", etc...
# The problem is that the only control there now is to either disable the dns_labels entirely (only valid if
# we don't use them), or change our label-prefix to conform to the very restricted form for the DNS labels
# These must be between 1 and 15 characters in length and can't include special characters or hyphens.
# They are discussing how to address that, but we were using the descriptive name style above for our cluster names
# and those now will be much less descriptive/scrunched...


# Retain only alphanumeric characters from the BRANCH_NAME and truncate
NEW_BRANCH=$(echo "$BRANCH_NAME" | sed 's/[^a-zA-Z0-9]//g')
NEW_BRANCH=${NEW_BRANCH:0:4}

CLUSTER_PREFIX="$NEW_BRANCH$BUILD_NUMBER"
if (( ${#CLUSTER_PREFIX} > 8 )); then
  CLUSTER_PREFIX=${CLUSTER_PREFIX:0:8}
fi

TIMESTAMP=${SHORT_TIME_STAMP}
if (( ${#TIMESTAMP} > 4 )); then
  TIMESTAMP=${TIMESTAMP:0:4}
fi

CLUSTER_PREFIX="$CLUSTER_PREFIX$TIMESTAMP"
echo "$CLUSTER_PREFIX"
