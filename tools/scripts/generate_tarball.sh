#!/bin/bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ ! -f "$1" ]; then
  echo "You must specify the images list BOM file as input"
  exit 1
fi

if [ ! -d "$2" ]; then
  echo "Please specify temp directory"
  exit 1
fi

if [ -f "$3" ]; then
  echo "Output file already exists, please specify a new filename"
  exit 1
fi

bomFile=$1
tmpDir=$2
outputFile=$3

ARGS=
if [ "${DRY_RUN}" == "true" ]; then
  ARGS="-d"
fi

${SCRIPT_DIR}/vz-registry-image-helper.sh -f ${outputFile} -l ${tmpDir} -b ${bomFile} ${ARGS}
