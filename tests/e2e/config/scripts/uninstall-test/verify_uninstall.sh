#!/bin/bash

# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

DIFF_FOUND=false

DIFF=$(diff ${SCRIPT_DIR}/pre-install-resources/default.txt ${SCRIPT_DIR}/post-uninstall-resources/default.txt | grep "^>")
echo "Remaining resources:"
echo $DIFF
echo "Expected resources:"
EXPECTED_RESOURCES=`cat $SCRIPT_DIR/expected-resources.txt`
echo $EXPECTED_RESOURCES

if [ "$DIFF" == "$EXPECTED_RESOURCES" ] ; then
  echo "expected resources found"
else
  DIFF_FOUND=true
fi

if [ "$DIFF_FOUND" == true ] ; then
  exit 1
fi
