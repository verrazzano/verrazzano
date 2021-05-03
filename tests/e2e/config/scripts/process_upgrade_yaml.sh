#!/bin/bash

# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# This script is used to add the version: field to the verrazzano custom resource .yaml file
# It is needed to test upgrade
VERSION=$1
CR_IN_FILE=$2
CR_OUT_FILE=$3

# Create the output file and add the version field, and spec: field if missing
cp $CR_IN_FILE $CR_OUT_FILE
if ! grep -q 'spec:' $CR_IN_FILE; then
  echo 'spec:' >> $CR_OUT_FILE
fi
echo ' v' $VERSION >> $CR_OUT_FILE
