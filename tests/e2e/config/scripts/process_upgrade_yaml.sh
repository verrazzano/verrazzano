#!/bin/bash

# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# This script is used to add the version: field to the verrazzano custom resource .yaml file
# It is needed to test upgrade
VERSION=$1
CR_FILE=$2
yq -i eval ".spec.version = \"v${VERSION}\"" ${CR_FILE}
