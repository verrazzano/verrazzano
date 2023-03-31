#!/bin/bash
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
set -u

echo """

Moving experimental CRDs (${EXPERIMENTAL_CRDS}) from ${CRD_PATH} to ${EXPERIMENTAL_CRD_PATH}

"""

echo "Ensuring ${EXPERIMENTAL_CRD_PATH} exists"
mkdir -p ${EXPERIMENTAL_CRD_PATH}

for crdfile in $(echo "${EXPERIMENTAL_CRDS}"); do
  fullpath=${CRD_PATH}/${crdfile}
  if [ -e ${fullpath} ]; then
    mv -v ${fullpath} ${EXPERIMENTAL_CRD_PATH}
  else
    echo "${fullpath} does not exist, skipping"
  fi
done
