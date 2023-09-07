#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes-objectstore.txt
if [ $? -ne  0 ] ; then
  echo "image-sizes.txt not found"
  oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes.txt
  if [ $? -eq 0 ] ; then
      exit
  fi
  echo "os object put --file image-sizes.txt failed"
  exit
fi

diff ${WORKSPACE}/image-sizes-objectstore.txt ${WORKSPACE}/image-sizes.txt > returnvalue.txt
if [ $? -eq 0 ] ; then
  rm returnvalue.txt
fi

