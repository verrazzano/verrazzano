#!/bin/bash -x

# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

if [ -z "$1" ] ; then
  echo "Please provide the directory containing the resources"
  exit 1
fi

SCRIPT_DIR="$1"
DIFF_FOUND=false

DIFF=$(diff ${SCRIPT_DIR}/pre-install-resources/default.txt ${SCRIPT_DIR}/post-uninstall-resources/default.txt | grep "^>")
echo "Remaining resources:"
echo $DIFF

if [ -z "$DIFF" ] ; then
  echo "No resources found as expected"
else
  DIFF_FOUND=true
fi

if [ "$DIFF_FOUND" == true ] ; then
  exit 1
fi

OAM_CRD=$(kubectl get crds | grep "core.oam.dev")
echo "Remaining core.oam.dev CRDs:"
echo $OAM_CRD
if [ -z "$OAM_CRD" ] ; then
  echo "No core.oam.dev CRDs found as expected"
else
  OAM_CRD_FOUND=true
fi
if [ "$OAM_CRD_FOUND" == true ] ; then
  exit 1
fi
