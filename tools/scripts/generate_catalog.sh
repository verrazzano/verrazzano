#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the catalog file as input"
  exit 1
fi
CATALOG_FILE=$1

if [ -z "$2" ]; then
  echo "You must specify the Version"
  exit 1
fi
VERRAZZANO_VERSION=$2

GENERATED_CATALOG_FILE=$3
if [ -z "${GENERATED_CATALOG_FILE}" ]; then
  echo "You must specify the catalog filename as output"
  exit 1
fi

cp ${CATALOG_FILE} ${GENERATED_CATALOG_FILE}

# Update the catalog file with the Verrazzano version
regex=".*:.*"
sed -i"" -e "s|VERRAZZANO_VERSION|${VERRAZZANO_VERSION}|g" ${GENERATED_CATALOG_FILE}
