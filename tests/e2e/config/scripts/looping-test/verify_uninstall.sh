#!/bin/bash -x

# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

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
