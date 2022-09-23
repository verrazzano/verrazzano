#!/usr/bin/env bash

#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

if [ -z "$1" ]; then
  echo "Verrazzano release version is required"
  exit 1
fi
VERSION=$1

# Remove prefix v from version
VERSION_NUM=${VERSION:1}

# Verrazzano distribution from 1.4.0 release replaces operator.yaml with verrazzano-platform-operator.yaml in release assets
VERSION_14=1.4.0

# Derive the file name
OPERATOR_YAML=$(echo ${VERSION_NUM} ${VERSION_14} | awk '{if ($1 < $2) print "operator.yaml"; else print "verrazzano-platform-operator.yaml"}')

# Derive yaml file
OPERATOR_YAML_FILE="https://github.com/verrazzano/verrazzano/releases/download/${VERSION}/${OPERATOR_YAML}"

echo $OPERATOR_YAML_FILE
