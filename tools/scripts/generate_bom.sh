#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the BOM file as input"
  exit 1
fi
BOM_FILE=$1

if [ -z "$2" ]; then
  echo "You must specify the Version"
  exit 1
fi
VERRAZZANO_VERSION=$2

VERRAZZANO_APPLICATION_OPERATOR_IMAGE=$3
if [ -z "${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}" ]; then
  echo "You must specify the Application Operator Image"
  exit 1
fi

VERRAZZANO_CLUSTER_OPERATOR_IMAGE=$4
if [ -z "${VERRAZZANO_CLUSTER_OPERATOR_IMAGE}" ]; then
  echo "You must specify the Cluster Operator Image Name"
  exit 1
fi

VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME=$5
if [ -z "${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME}" ]; then
  echo "You must specify the Platform Operator Image Name"
  exit 1
fi

IMAGE_TAG=$6
if [ -z "${IMAGE_TAG}" ]; then
  echo "You must specify the Image Tag"
  exit 1
fi

GENERATED_BOM_FILE=$7
if [ -z "${GENERATED_BOM_FILE}" ]; then
  echo "You must specify the BOM filename as output"
  exit 1
fi

cp ${BOM_FILE} ${GENERATED_BOM_FILE}

# Update the BOM file for the application operator and platform operator images.
# The application operator image can be supplied as the image or image:tag, if it is image only the same tag will be used for both operators
# The platform operator image and tag are supplied separately
regex=".*:.*"
if [[ ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} =~ $regex ]] ; then
  sed -i"" -e "s|VERRAZZANO_APPLICATION_OPERATOR_IMAGE|$(echo ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} | rev | cut -d / -f 1 | rev | cut -d : -f 1)|g" ${GENERATED_BOM_FILE}
  sed -i"" -e "s|VERRAZZANO_APPLICATION_OPERATOR_TAG|$(echo ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}:UNDEFINED | rev | cut -d / -f 1 | rev | cut -d : -f 2)|g" ${GENERATED_BOM_FILE}
else
  sed -i"" -e "s|VERRAZZANO_APPLICATION_OPERATOR_IMAGE|${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}|g" ${GENERATED_BOM_FILE}
  sed -i"" -e "s|VERRAZZANO_APPLICATION_OPERATOR_TAG|${IMAGE_TAG}|g" ${GENERATED_BOM_FILE}
fi
if [[ ${VERRAZZANO_CLUSTER_OPERATOR_IMAGE} =~ $regex ]] ; then
  sed -i"" -e "s|VERRAZZANO_CLUSTER_OPERATOR_IMAGE|$(echo ${VERRAZZANO_CLUSTER_OPERATOR_IMAGE} | rev | cut -d / -f 1 | rev | cut -d : -f 1)|g" ${GENERATED_BOM_FILE}
  sed -i"" -e "s|VERRAZZANO_CLUSTER_OPERATOR_TAG|$(echo ${VERRAZZANO_CLUSTER_OPERATOR_IMAGE}:UNDEFINED | rev | cut -d / -f 1 | rev | cut -d : -f 2)|g" ${GENERATED_BOM_FILE}
else
  sed -i"" -e "s|VERRAZZANO_CLUSTER_OPERATOR_IMAGE|${VERRAZZANO_CLUSTER_OPERATOR_IMAGE}|g" ${GENERATED_BOM_FILE}
  sed -i"" -e "s|VERRAZZANO_CLUSTER_OPERATOR_TAG|${IMAGE_TAG}|g" ${GENERATED_BOM_FILE}
fi
sed -i"" -e "s|VERRAZZANO_PLATFORM_OPERATOR_IMAGE|${VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME}|g" ${GENERATED_BOM_FILE}
sed -i"" -e "s|VERRAZZANO_PLATFORM_OPERATOR_TAG|${IMAGE_TAG}|g" ${GENERATED_BOM_FILE}
sed -i"" -e "s|VERRAZZANO_VERSION|${VERRAZZANO_VERSION}|g" ${GENERATED_BOM_FILE}
