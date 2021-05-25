#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the BOM file as input"
  exit 1
fi
BOM_FILE=$1

if [ -z "$2" ]; then
  echo "You must specify the Application Operator Image name"
  exit 1
fi
APP_OPERATOR_IMAGE_NAME=$2

if [ -z "$3" ]; then
  echo "You must specify the Platform Operator Image name"
  exit 1
fi
PLATFORM_OPERATOR_IMAGE_NAME=$3

if [ -z "$4" ]; then
  echo "You must specify the Image tag"
  exit 1
fi
IMAGE_TAG=$4

if [ -z "$5" ]; then
  echo "You must specify the BOM filename as output"
  exit 1
fi
GENERATED_BOM_FILE=$5

cp ${BOM_FILE} ${GENERATED_BOM_FILE}

# Update the BOM file for the application operator and platform operator images.
# The image names are expected to be supplied as the bare image name
# The tag is supplied explicitly
sed -i"" -e "s|VERRAZZANO_APPLICATION_OPERATOR_IMAGE|${APP_OPERATOR_IMAGE_NAME}|g" ${GENERATED_BOM_FILE}
sed -i"" -e "s|VERRAZZANO_APPLICATION_OPERATOR_TAG|${IMAGE_TAG}|g" ${GENERATED_BOM_FILE}
sed -i"" -e "s|VERRAZZANO_PLATFORM_OPERATOR_IMAGE|${PLATFORM_OPERATOR_IMAGE_NAME}|g" ${GENERATED_BOM_FILE}
sed -i"" -e "s|VERRAZZANO_PLATFORM_OPERATOR_TAG|${IMAGE_TAG}|g" ${GENERATED_BOM_FILE}
